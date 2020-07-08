// Copyright 2020 The Execstub Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipc

import (
	"context"
	"io"
	"log"
	"os"
	"syscall"

	"github.com/congop/execstub/internal/rand"
	"github.com/congop/execstub/pkg/comproto"
	fifo2 "github.com/containerd/fifo"
	"github.com/pkg/errors"
)

// StubbingComChannel enable inter process communication between test and test helper process.
type StubbingComChannel struct {
	StubberPipePath string
	stubberPipeRwc  io.ReadWriteCloser

	StubRequestChan  chan *comproto.StubRequest
	ExecResponseChan chan *comproto.ExecOutcome

	TestProcessHelperPipePath string
	testProcessHelperPipeRwc  io.ReadWriteCloser
}

// CleanUp performs clean up of channels and named pipes
func (comChannel *StubbingComChannel) CleanUp() {
	comChannel.Close()
	if comChannel.StubberPipePath != "" {
		os.Remove(comChannel.StubberPipePath)
		comChannel.StubberPipePath = ""
	}
	if comChannel.TestProcessHelperPipePath != "" {
		os.Remove(comChannel.TestProcessHelperPipePath)
		comChannel.TestProcessHelperPipePath = ""
	}
}

// Close closes all communication mechanisms this com-channel is using.
func (comChannel *StubbingComChannel) Close() {
	if comChannel.StubRequestChan != nil {
		close(comChannel.StubRequestChan)
	}
	if comChannel.ExecResponseChan != nil {
		close(comChannel.ExecResponseChan)
	}
	if comChannel.stubberPipeRwc != nil {
		comChannel.stubberPipeRwc.Close()
	}
	if comChannel.testProcessHelperPipeRwc != nil {
		comChannel.testProcessHelperPipeRwc.Close()
	}
}

// NewStubbingComChannel create a new com channel given the executable path.
func NewStubbingComChannel(path string) (comChannel *StubbingComChannel, err error) {
	randStr := rand.NextRandInt63AsHexStr()
	stubberPipePath := path + "_stubber_pipe_" + randStr

	if err := syscall.Mkfifo(stubberPipePath, 0777); nil != err {
		err = errors.Wrapf(
			err, "Could not create named pipe for stubber at: %s", stubberPipePath)
		return nil, err
	}

	testProcessHelperPipePath := path + "_testprocesshelper_pipe_" + randStr

	if err := syscall.Mkfifo(testProcessHelperPipePath, 0777); nil != err {
		err = errors.Wrapf(
			err,
			"Could not create named pipe for test process helper at: %s",
			stubberPipePath)
		return nil, err
	}
	comChannel = &StubbingComChannel{
		StubberPipePath:           stubberPipePath,
		TestProcessHelperPipePath: testProcessHelperPipePath,
		StubRequestChan:           make(chan *comproto.StubRequest, 8),
		ExecResponseChan:          make(chan *comproto.ExecOutcome, 8),
	}
	var writer io.ReadWriteCloser
	if writer, err = fifo2.OpenFifo(
		context.Background(),
		comChannel.StubberPipePath,
		syscall.O_RDWR, os.ModeNamedPipe); err != nil {
		err = errors.Wrapf(
			err,
			"error getting writer for pipe[stubberPipePath]: %s",
			comChannel.StubberPipePath)
		comChannel.CleanUp()
		return nil, err
	}
	comChannel.stubberPipeRwc = writer

	if writer, err =
		fifo2.OpenFifo(
			context.Background(),
			comChannel.TestProcessHelperPipePath,
			syscall.O_RDWR, os.ModeNamedPipe); err != nil {
		err = errors.Wrapf(err,
			"error getting writer for pipe[testProcessHelperPipePath]: %s",
			comChannel.TestProcessHelperPipePath)
		comChannel.CleanUp()
		return nil, err
	}
	comChannel.testProcessHelperPipeRwc = writer

	// hiding the named pipes behinds chan(s) so that client are not required
	// to deal with the channel specific technology
	go func() {
		for {
			resp := <-comChannel.ExecResponseChan
			if resp == nil {
				return
			}
			err := comChannel.sendResponseToTestProcess(resp)
			if err != nil {
				log.Printf(
					"Error sending response to test process: "+
						"\n\tresp=%v \n\terr:%s \n\terr:%##v",
					resp, err, err)
			}
		}
	}()

	go func() {
		for {
			req, errd := comproto.StubRequestDecoderFunc(comChannel.stubberPipeRwc)
			if nil != errd {
				log.Printf(
					"Failed to read from test process pipe:%s, err:%v",
					stubberPipePath, errd)
				return
			}

			comChannel.StubRequestChan <- req
			log.Printf("Stubbing resquest queued:%#v", req)
		}
	}()

	return comChannel, nil
}

func (comChannel *StubbingComChannel) sendResponseToTestProcess(response *comproto.ExecOutcome) error {
	execEncoderFunc := comproto.ExecOutcomeEncoderFunc(comChannel.testProcessHelperPipeRwc)
	err := execEncoderFunc(response)
	log.Printf("Stubbing ExecOutcome send to test process resp=%#v err:%v", response, err)
	return err
}
