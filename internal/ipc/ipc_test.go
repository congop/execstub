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
	"math"
	"os"
	"testing"
	"time"

	"github.com/congop/execstub/internal/fifo"
	"github.com/congop/execstub/pkg/comproto"
	"github.com/stretchr/testify/assert"
)

func Test_newPipeComChannel(t *testing.T) {
	testExecPath := os.Args[0]
	timeout := time.Duration(10 * time.Second)
	stubberPipePath, testProcessHelperPipePath := fifo.NewFifoNamesForIpc(testExecPath)
	///
	comChannel, err := NewStubbingComChannel(stubberPipePath, testProcessHelperPipePath)
	t.Cleanup(comChannel.CleanUp)
	///
	if err != nil {
		t.Fatalf("New Com Channel should not have failed:%v", err)
	}

	req := comproto.StubRequest{
		CmdName: "BlablaExe",
		Key:     "BlaBla_fffzzz_666444333",
		Args:    []string{"arg1", "argb"},
	}

	err = comproto.WriteStubRequestToNamedPipe(comChannel.StubberPipePath, req, timeout)
	if err != nil {
		t.Fatalf("sendRequestStubRequest failed: %##v", err)
	}
	timeoutChan := make(chan bool, 8)

	timeoutAufterFunc := time.AfterFunc(time.Duration(timeout), func() { timeoutChan <- true })
	defer timeoutAufterFunc.Stop()

	var reqRead *comproto.StubRequest
	select {
	case reqRead = <-comChannel.StubRequestChan:
	case <-timeoutChan:
		t.Fatalf("Timeout could not read request from pipe")
	}

	assert.Equal(t, reqRead, &req)

	response := comproto.ExecOutcome{
		InternalErrTxt: "errtxt1",
		ExitCode:       math.MaxInt8,
		Key:            "BlaBla_fffzzz_666444333",
		Stderr:         "stderr0",
		Stdout:         "stdout2",
	}

	comChannel.ExecResponseChan <- &response
	stubbingOutcome, err := comproto.ReadStubbingResponseFromNamedPipe(comChannel.TestProcessHelperPipePath, time.Duration(10*time.Second))
	if err != nil {
		t.Fatalf("The outcome send through chan should have been read error free but had error(%s):%##v", err, err)
	}
	assert.Equal(t, *stubbingOutcome, response)
}
