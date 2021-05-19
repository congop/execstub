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

package runtime

import (
	"fmt"
	"os"
	"strings"
)

var execExts []string

func init() {
	if IsWindows() {
		// PATHEXT=.COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC
		execExts = getFileExts()
	} else {
		execExts = nil
	}
}

func getFileExts() []string {
	pathExt, ok := os.LookupEnv("PATHEXT")
	if !ok {
		return nil
	}
	return strings.Split(strings.ToLower(pathExt), ";")
}

func AssertFileNameEndsWithExecExt(fn string) {
	if !IsWindows() {
		return
	}
	if len(execExts) == 0 {
		return
	}
	fnLowerCase := strings.ToLower(fn)
	for _, ext := range execExts {
		if strings.HasSuffix(fnLowerCase, ext) {
			return
		}
	}
	mesg := fmt.Sprintf(
		"file name (%s) must end with a known executable extension[%v]",
		fn, execExts)
	panic(mesg)
}

func EnsureHasExecExt(fn string) string {
	if !IsWindows() {
		return fn
	}
	fnLowerCase := strings.ToLower(fn)
	for _, ext := range execExts {
		if strings.HasSuffix(fnLowerCase, ext) {
			return fn
		}
	}
	return fn + ".exe"
}
