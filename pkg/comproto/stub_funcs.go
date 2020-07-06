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
	"fmt"
	"math"
	"sync/atomic"
)

// StubFunc function use to produce the outcome for a stubbed command execution
type StubFunc func(sreq StubRequest) *ExecOutcome

// RecordingExecutions adds request recording feature to the given StubFunc
func RecordingExecutions(
	stubFunc StubFunc,
) (recordingStubFunc StubFunc, reqsStore *[]StubRequest) {
	sreqs := make([]StubRequest, 0)

	recordingStubFunc = func(sreq StubRequest) *ExecOutcome {
		sreqs = append(sreqs, sreq)
		return stubFunc(sreq)
	}
	reqsStore = &sreqs
	return
}

// AdaptOutcomeToCmdStub returns a StubFunc which will return the given outcome.
func AdaptOutcomeToCmdStub(
	outcome *ExecOutcome,
) func(sreq StubRequest) *ExecOutcome {
	return func(sreq StubRequest) *ExecOutcome {
		return outcome
	}
}

// AdaptOutcomesToCmdStub returns a CmdStub which will yield the given outcomes.
func AdaptOutcomesToCmdStub(
	outcomes []*ExecOutcome,
	repeatLast bool,
) func(sreq StubRequest) *ExecOutcome {
	funcs := make([]StubFunc, 0, len(outcomes))
	if len(outcomes) > 0 {
		for _, o := range outcomes {
			funcs = append(funcs, AdaptOutcomeToCmdStub(o))
		}
	}
	return AdaptFuncsToCmdStub(funcs, repeatLast)
}

//AdaptFuncsToCmdStub returns a CmdStub which will use a list of function to procudes the execution outcomes.
func AdaptFuncsToCmdStub(
	stubFuncs []StubFunc,
	repeatLast bool,
) func(sreq StubRequest) *ExecOutcome {
	next := new(int32)
	*next = 0
	return func(sreq StubRequest) *ExecOutcome {
		index := int(*next)
		defer func() { atomic.AddInt32(next, 1) }()
		if index >= len(stubFuncs) {
			if !repeatLast {
				internalErrTxt := fmt.Sprintf(
					"Too many executions while repeat-last not selected: "+
						"max=%d, current request count=%d",
					len(stubFuncs), index+1)
				return &ExecOutcome{
					Key:            sreq.Key,
					ExitCode:       math.MaxUint8,
					InternalErrTxt: internalErrTxt,
				}
			}
			index = len(stubFuncs) - 1
		}
		stubFunc := stubFuncs[index]

		return stubFunc(sreq)
	}
}
