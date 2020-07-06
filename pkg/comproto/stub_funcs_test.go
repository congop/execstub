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
	"reflect"
	"testing"
)

func TestAdaptFuncsToCmdStub(t *testing.T) {
	type args struct {
		stubFuncs  []StubFunc
		repeatLast bool
	}
	tests := []struct {
		name string
		args args
		want []ExecOutcome
	}{
		{
			name: "Should sequencially call the funcs and repeate the last",
			args: args{
				stubFuncs: []StubFunc{
					func(sreq StubRequest) *ExecOutcome {
						return &ExecOutcome{
							Key: "1", ExitCode: 0, Stderr: "err1",
							Stdout: "o1", InternalErrTxt: "ie1",
						}
					},
					func(sreq StubRequest) *ExecOutcome {
						return &ExecOutcome{
							Key: "2", ExitCode: 2, Stderr: "err2",
							Stdout: "o2", InternalErrTxt: "ie2",
						}
					},
				},
				repeatLast: true,
			},
			want: []ExecOutcome{
				{
					Key: "1", ExitCode: 0, Stderr: "err1",
					Stdout: "o1", InternalErrTxt: "ie1",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
			},
		},
		{
			name: "Should sequencially call the funcs and return error txt after func exaustion",
			args: args{
				stubFuncs: []StubFunc{
					func(sreq StubRequest) *ExecOutcome {
						return &ExecOutcome{
							Key: "1", ExitCode: 0, Stderr: "err1",
							Stdout: "o1", InternalErrTxt: "ie1",
						}
					},
					func(sreq StubRequest) *ExecOutcome {
						return &ExecOutcome{
							Key: "2", ExitCode: 2, Stderr: "err2",
							Stdout: "o2", InternalErrTxt: "ie2",
						}
					},
				},
				repeatLast: false,
			},
			want: []ExecOutcome{
				{
					Key: "1", ExitCode: 0, Stderr: "err1",
					Stdout: "o1", InternalErrTxt: "ie1",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "", ExitCode: 255, Stderr: "", Stdout: "",
					InternalErrTxt: "Too many executions while repeat-last not selected: max=2, current request count=3",
				},
				{
					Key: "", ExitCode: 255, Stderr: "", Stdout: "",
					InternalErrTxt: "Too many executions while repeat-last not selected: max=2, current request count=4",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sreq := StubRequest{}
			got := make([]ExecOutcome, 0, 4)
			///
			stubFunc := AdaptFuncsToCmdStub(tt.args.stubFuncs, tt.args.repeatLast)
			for i := 0; i < 4; i++ {
				got = append(got, *stubFunc(sreq))
			}
			///

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdaptToCmdStubFuncs() \ngot =%v, \nwant=%v", got, tt.want)
			}
		})
	}
}

func TestAdaptOutcomessToCmdStub(t *testing.T) {
	type args struct {
		outcomes   []*ExecOutcome
		repeatLast bool
	}
	tests := []struct {
		name string
		args args
		want []ExecOutcome
	}{
		{
			name: "Should sequencially call the funcs and repeate the last",
			args: args{
				outcomes: []*ExecOutcome{
					{
						Key: "1", ExitCode: 0, Stderr: "err1",
						Stdout: "o1", InternalErrTxt: "ie1",
					},
					{
						Key: "2", ExitCode: 2, Stderr: "err2",
						Stdout: "o2", InternalErrTxt: "ie2",
					},
				},
				repeatLast: true,
			},
			want: []ExecOutcome{
				{
					Key: "1", ExitCode: 0, Stderr: "err1",
					Stdout: "o1", InternalErrTxt: "ie1",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
			},
		},
		{
			name: "Should sequencially call the funcs and return error txt after func exaustion",
			args: args{
				outcomes: []*ExecOutcome{
					{
						Key: "1", ExitCode: 0, Stderr: "err1",
						Stdout: "o1", InternalErrTxt: "ie1",
					},
					{
						Key: "2", ExitCode: 2, Stderr: "err2",
						Stdout: "o2", InternalErrTxt: "ie2",
					},
				},
				repeatLast: false,
			},
			want: []ExecOutcome{
				{
					Key: "1", ExitCode: 0, Stderr: "err1",
					Stdout: "o1", InternalErrTxt: "ie1",
				},
				{
					Key: "2", ExitCode: 2, Stderr: "err2",
					Stdout: "o2", InternalErrTxt: "ie2",
				},
				{
					Key: "", ExitCode: 255, Stderr: "", Stdout: "",
					InternalErrTxt: "Too many executions while repeat-last not selected: max=2, current request count=3",
				},
				{
					Key: "", ExitCode: 255, Stderr: "", Stdout: "",
					InternalErrTxt: "Too many executions while repeat-last not selected: max=2, current request count=4",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sreq := StubRequest{}
			got := make([]ExecOutcome, 0, 4)
			///
			stubFunc := AdaptOutcomesToCmdStub(tt.args.outcomes, tt.args.repeatLast)
			for i := 0; i < 4; i++ {
				got = append(got, *stubFunc(sreq))
			}
			///

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AdaptToCmdStubFuncs() \ngot =%v, \nwant=%v", got, tt.want)
			}
		})
	}
}

func TestRecordingExecutions(t *testing.T) {
	type args struct {
		stubFunc StubFunc
		reqs     []StubRequest
	}
	tests := []struct {
		name          string
		args          args
		wantReqsStore []StubRequest
	}{
		{
			name: "Should record requests",
			args: args{
				stubFunc: func(sreq StubRequest) *ExecOutcome { return nil },
				reqs: []StubRequest{
					{CmdName: "c1", Args: []string{"a1", "a2"}, Key: "k1"},
					{CmdName: "c2", Args: []string{"a21", "a22"}, Key: "k2"},
				},
			},
			wantReqsStore: []StubRequest{
				{CmdName: "c1", Args: []string{"a1", "a2"}, Key: "k1"},
				{CmdName: "c2", Args: []string{"a21", "a22"}, Key: "k2"},
			},
		},

		{
			name: "Should record no requests",
			args: args{
				stubFunc: func(sreq StubRequest) *ExecOutcome { return nil },
				reqs:     []StubRequest{},
			},
			wantReqsStore: []StubRequest{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRecordingStubFunc, gotReqsStore := RecordingExecutions(tt.args.stubFunc)
			for _, req := range tt.args.reqs {
				gotRecordingStubFunc(req)
			}

			if !reflect.DeepEqual(*gotReqsStore, tt.wantReqsStore) {
				t.Errorf("RecordingExecutions() \ngot = %v \nwant= %v", *gotReqsStore, tt.wantReqsStore)
			}
		})
	}
}
