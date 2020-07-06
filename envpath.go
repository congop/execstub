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
	"os"
	"strings"
)

type envPath struct {
	parts []string
}

func (epath envPath) String() string {
	return strings.Join(epath.parts, envPathSeparator())
}

func envPathSeparator() string {
	return string(os.PathListSeparator)
}

func currentEnvPath() *envPath {
	path := os.Getenv("PATH")
	return newEnvPath(path)
}

func newEnvPath(envPathAsString string) *envPath {
	epath := envPath{}
	return epath.positionFirst(envPathAsString)
}

func (epath *envPath) positionFirst(partsAsString string) (ep *envPath) {
	splits := strings.Split(strings.TrimSpace(partsAsString), envPathSeparator())
	newParts := make([]string, 0, len(splits)+len(epath.parts))
	appendedMap := map[string]string{}
	for _, split := range splits {
		split = strings.TrimSpace(split)
		if split == "" {
			continue
		}
		if _, ok := appendedMap[split]; ok {
			continue
		}
		newParts = append(newParts, split)
		appendedMap[split] = ""
	}
	for _, part := range epath.parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := appendedMap[part]; ok {
			continue
		}
		newParts = append(newParts, part)
		appendedMap[part] = ""
	}
	epath.parts = newParts
	return epath
}

func (epath *envPath) removeParts(partsAsString string) *envPath {
	splits := strings.Split(strings.TrimSpace(partsAsString), envPathSeparator())
	appendedMap := map[string]string{}
	for _, split := range splits {
		appendedMap[split] = ""
	}
	newPaths := make([]string, 0, len(epath.parts))
	for _, part := range epath.parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := appendedMap[part]; ok {
			continue
		}
		newPaths = append(newPaths, part)
		appendedMap[part] = ""
	}
	epath.parts = newPaths
	return epath
}
