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
	"sync"
	"syscall"
	"time"

	"github.com/congop/execstub/internal/fifo"
	"github.com/congop/execstub/pkg/comproto"
	"github.com/pkg/errors"
)

// ErrChannelClosed used to signal that the channel has been closed.
var ErrChannelClosed = errors.New("Channel is closed")

// StubbingComChannel enable inter process communication between test and test helper process.
type StubbingComChannel struct {
	StubberPipePath string
	stubberPipeRwc  io.ReadWriteCloser

	StubRequestChan       chan *comproto.StubRequest
	StubRequestChanMutex  sync.Mutex
	ExecResponseChan      chan *comproto.ExecOutcome
	ExecResponseChanMutex sync.Mutex

	TestProcessHelperPipePath string
	testProcessHelperPipeRwc  io.ReadWriteCloser

	wg sync.WaitGroup
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

func (comChannel *StubbingComChannel) enqueueRequest(req *comproto.StubRequest) error {
	comChannel.StubRequestChanMutex.Lock()
	defer comChannel.StubRequestChanMutex.Unlock()
	if comChannel.StubRequestChan == nil {
		return ErrChannelClosed
	}
	comChannel.StubRequestChan <- req
	log.Printf("Stubbing resquest queued:%#v", req)
	return nil
}

func (comChannel *StubbingComChannel) EnqueueRespnse(resp *comproto.ExecOutcome) error {
	comChannel.ExecResponseChanMutex.Lock()
	defer comChannel.ExecResponseChanMutex.Unlock()
	if comChannel.ExecResponseChan == nil {
		return ErrChannelClosed
	}
	comChannel.ExecResponseChan <- resp
	log.Printf("Response queued:%#v", resp)
	return nil
}

func (comChannel *StubbingComChannel) closeStubResquestChan() {
	comChannel.StubRequestChanMutex.Lock()
	defer comChannel.StubRequestChanMutex.Unlock()
	if comChannel.StubRequestChan != nil {
		close(comChannel.StubRequestChan)
	}
}

func (comChannel *StubbingComChannel) closeExecResponseChan() {
	comChannel.ExecResponseChanMutex.Lock()
	defer comChannel.ExecResponseChanMutex.Unlock()
	if comChannel.ExecResponseChan != nil {
		close(comChannel.ExecResponseChan)
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration, timeoutMsg string) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		// TODO this could block forever leaking the go routine.
		// maybe do: wg.Done() until panic then recover on timeout?
		wg.Wait()
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		log.Print(timeoutMsg)
		return
	}
}

// Close closes all communication mechanisms this com-channel is using.
func (comChannel *StubbingComChannel) Close() {
	closeReq := comproto.StopOperationRequest()
	_ = comproto.WriteStubRequestToNamedPipe(comChannel.StubberPipePath, *closeReq, time.Second)

	comChannel.closeStubResquestChan()
	comChannel.closeExecResponseChan()

	waitTimeout(&comChannel.wg, 2*time.Second, "Timeout waiting channel activities to stop")

	if comChannel.stubberPipeRwc != nil {
		comChannel.stubberPipeRwc.Close()
	}
	if comChannel.testProcessHelperPipeRwc != nil {
		comChannel.testProcessHelperPipeRwc.Close()
	}
}

// NewStubbingComChannel create a new com channel given the executable path.
func NewStubbingComChannel(stubberPipePath string, testProcessHelperPipePath string) (comChannel *StubbingComChannel, err error) {
	if err := fifo.Mkfifo(stubberPipePath, 0770); nil != err {
		err = errors.Wrapf(
			err, "Could not create named pipe for stubber at: %s", stubberPipePath)
		return nil, err
	}

	if err := fifo.Mkfifo(testProcessHelperPipePath, 0770); nil != err {
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
	if writer, err = fifo.OpenFifo(
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
	// syscall.O_WRONLY
	if writer, err =
		fifo.OpenFifo(
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
	comChannel.wg.Add(2)
	go func() {
		defer comChannel.wg.Done()
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
		defer comChannel.wg.Done()
		for {
			req, errd := comproto.StubRequestDecoderFunc(comChannel.stubberPipeRwc)
			log.Printf("StubRequest from stubberPipe:%#v", req)
			if nil != errd {
				log.Printf(
					"Failed to read from test process pipe:%s, err:%v",
					stubberPipePath, errd)
				return
			}
			if req.IsRequestingStop() {
				log.Printf("Stop reading stubberPipeRwc on reqest:%#v", req)
				return
			}
			errs := comChannel.enqueueRequest(req)
			if errs != nil {
				return
			}
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
