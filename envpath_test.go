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

package execstub

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func Test_currentEnvPath(t *testing.T) {
	want := os.Getenv("PATH")
	///
	got := currentEnvPath()
	gotAsStr := fmt.Sprint(got)
	///

	wantDeduplicated := deduplicatePath(want, envPathSeparator())
	if gotAsStr != wantDeduplicated {
		t.Errorf(
			"currentEnvPath() \ngotStr= %s \ngot   = %#v \nwant  = %s",
			gotAsStr, got, want)
	}
}

func deduplicatePath(pathAsString, separator string) (deduplicatedPath string) {
	splits := strings.Split(strings.TrimSpace(pathAsString), separator)
	deduplicatedPathParts := make([]string, 0, len(splits))
	alreadyAdded := map[string]string{}
	for _, split := range splits {
		if _, ok := alreadyAdded[split]; ok {
			continue
		}
		deduplicatedPathParts = append(deduplicatedPathParts, split)
		alreadyAdded[split] = ""
	}
	return strings.Join(deduplicatedPathParts, separator)
}
