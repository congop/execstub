// +build ignore

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

package assets

import "encoding/base64"

// ExecStubExeHash returns the hash of the bash script that can be use to stub an executable.
// This is here mainly to aid detect changes without having to check be binary in base64.
func ExecStubBashScriptHash() string {
	hash := "{{ .ExecStubHash }}"
	return hash
}

// ExecStubBashScript return a bash script that can be used to stub an executable.
func ExecStubBashScript() (script []byte, err error) {
	execStubBase64 := "{{ .ExecStubBase64 }}"
	return base64.StdEncoding.DecodeString(execStubBase64)
}
