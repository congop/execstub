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

package comproto

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"syscall"
	"time"

	fifo2 "github.com/containerd/fifo"
	"github.com/pkg/errors"
)

// StubbingOngoing tells whether the current process is run as stub
// instead of an actual execution.
// The check is based of off process environment
func StubbingOngoing() bool {
	return os.Getenv("__EXECSTUBBING_GO_WANT_HELPER_PROCESS") == "1"
}

// EffectuateAlternativeExecOutcome effectuates an alternative
// outcome to the configured one for test helper processes.
func EffectuateAlternativeExecOutcome(stubFunc StubFunc) {
	if !StubbingOngoing() {
		return
	}
	cfg, req := cmdConfigWithRequestOrExitOnError()

	err := NewStubRequestDirRepo(cfg.DataDir).Save(req)
	if nil != err {
		fmt.Fprintf(os.Stderr, "fail to save request[%#v]; err=%v", req, err)
		os.Exit(math.MaxUint8)
	}

	if nil == stubFunc {
		fmt.Fprint(os.Stderr, "stubFunc must not be nil when overrinding outcome")
		os.Exit(math.MaxUint8)
	}

	outcome := stubFunc(req)
	if outcome.Stderr != "" {
		fmt.Fprint(os.Stderr, outcome.Stderr)
	}
	if outcome.Stdout != "" {
		fmt.Fprint(os.Stdout, outcome.Stdout)
	}
	if outcome.InternalErrTxt != "" {
		fmt.Fprint(os.Stderr, outcome.InternalErrTxt)
		os.Exit(math.MaxUint8)
	}
	os.Exit(int(outcome.ExitCode))
}

// EffectuateConfiguredExecOutcome effectuates the configured static or
// dynamic outcome for test helper processes.
func EffectuateConfiguredExecOutcome(
	extraJobOnStubRequestForStaticMode func(StubRequest) error,
) {
	if !StubbingOngoing() {
		return
	}

	cfg, req := cmdConfigWithRequestOrExitOnError()

	if cfg.UseStaticOutCome() {
		effectuateStaticOutcome(*cfg, extraJobOnStubRequestForStaticMode, req)
	}

	pipePathStubber := cfg.PipeStubber
	pipePathTestHelperProc := cfg.PipeTestHelperProcess
	timeout := cfg.TimeoutAsDurationOrDefault()
	exitCode := EffectuateDynamicOutcome(
		timeout, pipePathStubber, pipePathTestHelperProc,
		req, os.Stderr, os.Stdout)
	defer os.Exit(int(exitCode))
}

func cmdConfigWithRequestOrExitOnError() (*CmdConfig, StubRequest) {
	stubCmdConfigPath := os.Getenv("__EXECSTUBBING_STUB_CMD_CONFIG")
	cfg, err := CmdConfigLoadedFromFile(stubCmdConfigPath)
	if err != nil {
		fmt.Printf(
			"could not load CmdConfig from \nfile:%s \nerr=%v",
			stubCmdConfigPath, err)
		os.Exit(math.MaxUint8)
	}
	req := cfg.StubRequestWith(actualCmdArgsFromTestHelpProcess())
	return cfg, req
}

// actualCmdArgsFromTestHelpProcess return the actual command argument
// given test process execution
// testexe -test.run="pattern" -- arg1 arg2 .. argn.
func actualCmdArgsFromTestHelpProcess() []string {
	if len(os.Args) <= 3 {
		return []string{}
	}
	return os.Args[3:]
}

func doExtraJobOnStubRequest(
	extraJobOnStubRequest func(StubRequest) error,
	req StubRequest,
) (err error) {
	if extraJobOnStubRequest == nil {
		return nil
	}

	hasNotPanicked := false
	defer func() {
		if r := recover(); r != nil || !hasNotPanicked {
			err = errors.Errorf(
				"panic while doing extra job on stubbing request[%#v], %##v", req, r)
			return
		}
	}()
	err = extraJobOnStubRequest(req)
	hasNotPanicked = true
	if err != nil {
		err = errors.Wrapf(
			err, "fail to do extra job on stubbing request[%#v]", req)
		return err
	}
	return nil
}

// effectuateStaticOutcome realizes the configured static outcome.
func effectuateStaticOutcome(
	cfg CmdConfig,
	extraJobOnStubRequest func(StubRequest) error,
	req StubRequest,
) {
	err := NewStubRequestDirRepo(cfg.DataDir).Save(req)
	if nil != err {
		fmt.Fprint(os.Stderr,
			"will not effectuate static outcome because failed at saving request:",
			err)
		os.Exit(math.MaxUint8)
	}

	err = doExtraJobOnStubRequest(extraJobOnStubRequest, req)
	if err != nil {
		fmt.Printf("fail to effectuate static outcome err=%v", err)
		os.Exit(math.MaxUint8)
	}

	if cfg.StderrAvail() {
		fmt.Fprint(os.Stderr, cfg.TxtStderr)
	}
	if cfg.StdoutAvail() {
		fmt.Fprint(os.Stdout, cfg.TxtStdout)
	}
	exitCode, err := cfg.ExitCodeUint8()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(math.MaxUint8)
	}

	defer os.Exit(int(exitCode))
}

// EffectuateDynamicOutcome uses IPC to request execution outcome from running test.
func EffectuateDynamicOutcome(
	timeout time.Duration,
	pipePathStubber string,
	pipePathTestHelperProc string,
	req StubRequest,
	stderr io.Writer,
	stdout io.Writer,

) (exitCode uint8) {

	err := WriteStubRequestToNamedPipe(pipePathStubber, req, timeout)
	if err != nil {
		fmt.Fprintf(stderr,
			"Error while writing stubbing request to stubber named pipe[%s]running stub:%v // %#v",
			pipePathStubber, err, err)
		return math.MaxUint8
	}

	outcome, err := ReadStubbingResponseFromNamedPipe(pipePathTestHelperProc, timeout)
	if err != nil {
		fmt.Fprintf(stderr,
			"error while reading stubing outcome from test helper named pipe[%s]running stub:"+
				" \n\ttimeoutMillis=%d \n\terr=%v",
			pipePathTestHelperProc, timeout/time.Microsecond, err)
		return math.MaxUint8
	}

	if outcome.StdoutAvail() {
		fmt.Fprint(stdout, outcome.Stdout)
	}
	if outcome.StderrAvail() {
		fmt.Fprint(stderr, outcome.Stderr)
	}

	if outcome.InternalErrTxtAvail() {
		fmt.Fprint(stderr, outcome.InternalErrTxt)
		return math.MaxUint8
	}

	return outcome.ExitCode
}

// ReadStubbingResponseFromNamedPipe reads the stubbing outcome from the named pipe.
func ReadStubbingResponseFromNamedPipe(pipePath string, timeout time.Duration) (*ExecOutcome, error) {
	timeoutChan := make(chan bool, 1)
	outcomeChan := make(chan interface{}, 1)

	go func() {
		// go-func because both fifo2.OpenFifo(..) and dec.Decode(..) may block
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		reader, err := fifo2.OpenFifo(ctx, pipePath, syscall.O_RDONLY, os.ModeNamedPipe)
		cancel()
		if err != nil {
			outcomeChan <- errors.Wrapf(err, "error getting reader for named pipe[%s]: err:%v", pipePath, err)
			return
		}
		defer reader.Close()
		outcome, err := ExecOutcomeDecoderFunc(reader)

		if err != nil {
			outcomeChan <- errors.Wrapf(err, "Error decoding data as stubbing outcome from named pipe[%s]: err=%#v", pipePath, err)
		}
		outcomeChan <- *outcome
	}()

	timeoutAfterFunc := time.AfterFunc(timeout, func() { timeoutChan <- true })
	defer timeoutAfterFunc.Stop()
	select {
	case <-timeoutChan:
		return nil, errors.Errorf("Timeout while reading from named pipe[%s]", pipePath)
	case outcome, ok := <-outcomeChan:
		if !ok {
			return nil, errors.Errorf("Error reading stubbing outcome from channel holding data from named pipe[%s]", pipePath)
		}
		if err, ok := outcome.(error); ok {
			return nil, err
		}
		stubbingOutcome, ok := outcome.(ExecOutcome)
		if !ok {
			return nil, errors.Errorf(
				"Unsupported outcome type found in named pipe[%s]: %#v",
				pipePath, outcome)
		}
		return &stubbingOutcome, nil
	}
}

// WriteStubRequestToNamedPipe write the stubbing request to the named pipe
func WriteStubRequestToNamedPipe(
	pipePath string, stubReq StubRequest, timeout time.Duration,
) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	writer, err := fifo2.OpenFifo(ctx, pipePath, syscall.O_WRONLY, os.ModeNamedPipe)
	cancel()
	if err != nil {
		return errors.Wrapf(err,
			"Error getting writer for named pipe[%s]: err:%v", pipePath, err)
	}
	defer writer.Close()
	encoderFunc := StubRequestEncoderFunc(writer)
	err = encoderFunc((&stubReq))
	if err != nil {
		return errors.Wrapf(err,
			"Error writing gob encoded data to named pipe[%s]: err:%v",
			pipePath, err)
	}
	return nil
}
