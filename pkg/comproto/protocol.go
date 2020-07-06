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

// StubRequest holds data send to a stubber through a ComChannel to request stubbing
type StubRequest struct {
	// Key identifies the stubbing setup
	Key     string
	CmdName string
	Args    []string
}

// ExecOutcome modells what happens when the stubbed command is executed.
// It basically the content of std-out and std-err and the error code
// the stubbed command us supposed to yield.
// The stubber may encounter some internal err.
// In that case:
// - InternalErrTxt is provided.
// - ExitCode does not hold a meaningfull value
// - The requesting side should not use it as the expected exit code
// Not using an typed error here in order to ease serialization during ipc
// and allow usage both as dto and stubbing function return type.
type ExecOutcome struct {
	Key string

	// Expected command exec outcome
	// the receiving side is expected forward them as is
	Stdout   string
	Stderr   string
	ExitCode uint8

	InternalErrTxt string
}

// StderrAvail return true if Stderr is available (not "") and false otherwise.
func (o ExecOutcome) StderrAvail() bool {
	return o.Stderr != ""
}

// StdoutAvail return true if Stdout is available (not "") and false otherwise.
func (o ExecOutcome) StdoutAvail() bool {
	return o.Stdout != ""
}

// InternalErrTxtAvail return true if InternalErrTxt is available (not "") and false otherwise.
func (o ExecOutcome) InternalErrTxtAvail() bool {
	return o.InternalErrTxt != ""
}
