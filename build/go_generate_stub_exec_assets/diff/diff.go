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

package diff

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func isLineCommentOrEmpty(line string) bool {
	return line == "" || strings.HasPrefix(line, "//")
}

// equalIgnoreLineComment returns true both are either empty or line comment
// or are equal.
func equalIgnoreLineComment(_line1, _line2 string) bool {
	l1 := strings.TrimSpace(_line1)
	l2 := strings.TrimSpace(_line2)
	if isLineCommentOrEmpty(l1) && isLineCommentOrEmpty(l2) {
		return true
	}
	if len(l1) != len(l2) {
		return false
	}
	return l1 == l2
}

// EqualIgnoreLineCommentReader return true if the content delivered by the
// given readers are eq if line comments are ignored; false otherwise.
// Note that some empty line may be ignored too.
func EqualIgnoreLineCommentReader(src1, src2 io.Reader) bool {
	rsrc1 := bufio.NewScanner(src1)
	rsrc2 := bufio.NewScanner(src2)

	for {
		// we need to scan until both sources are done because the remaining
		// lines in the longer source may all be empty or line comments
		line1 := ""
		hasLineInSrc1 := rsrc1.Scan()
		if hasLineInSrc1 {
			line1 = rsrc1.Text()
		}

		line2 := ""
		hasLineInSrc2 := rsrc2.Scan()
		if hasLineInSrc2 {
			line2 = rsrc2.Text()
		}
		if !equalIgnoreLineComment(line1, line2) {
			return false
		}
		if !hasLineInSrc1 && !hasLineInSrc2 {
			break
		}
	}
	return true
}

// EqualIgnoreLineCommentPath return true if the content file specified by
// the given paths are equal if line comments are ignored; false otherwise.
// Note that some empty line may be ignored too.
func EqualIgnoreLineCommentPath(src1Path, src2Path string) (bool, error) {
	if src1Path == "" && src2Path == "" {
		return true, nil
	}
	src1, err := toReadCloser(src1Path)
	if err != nil {
		return false, err
	}
	defer src1.Close()

	src2, err := toReadCloser(src2Path)
	if err != nil {
		return false, err
	}
	defer src2.Close()

	eq := EqualIgnoreLineCommentReader(src2, src1)
	return eq, nil
}

func toReadCloser(path string) (io.ReadCloser, error) {
	if path == "" {
		return emptyBufReaderCloser(), nil
	}
	file, err := os.Open(path)

	if err != nil {
		if os.IsNotExist(err) {
			return emptyBufReaderCloser(), nil
		}
		err = errors.Wrapf(err, "fail to open file[%s]", path)
		return nil, err
	}
	return file, err
}

func emptyBufReaderCloser() io.ReadCloser {
	var buf bytes.Buffer
	return ioutil.NopCloser(&buf)
}
