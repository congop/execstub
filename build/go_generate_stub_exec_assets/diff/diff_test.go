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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func Test_hasDiffWhenIgnroingLineComments(t *testing.T) {
	type args struct {
		c1, c2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty content are equal",
			args: args{
				c1: "",
				c2: "",
			},
			want: true,
		},
		{
			name: "different single line are not equal",
			args: args{
				c1: "line1",
				c2: "line2",
			},
			want: false,
		},
		{
			name: "different single comment line are equal",
			args: args{
				c1: " // line1",
				c2: "// line2   ",
			},
			want: true,
		},

		{
			name: "multiline are equal if their actual diff are line comments",
			args: args{
				c1: ` // line1 
						lin2
						
						//l3
						lin4
						`,
				c2: ` // line1 
						lin2

						//l3
						lin4
						// comment1
						`,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src1 := strings.NewReader(tt.args.c1)
			src2 := strings.NewReader(tt.args.c2)
			if got := EqualIgnoreLineCommentReader(src1, src2); got != tt.want {
				t.Errorf(
					"hasDiffWhenIgnroingLineComments() = %v, want %v \n\tline1:%s \n\tline2=%s",
					got, tt.want, tt.args.c1, tt.args.c2)
			}
		})
	}
}

func TestHasLinewiseDiffWhenIgnroingLineCommentsWithPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "diff-linewise-*")
	if err != nil {
		t.Errorf("could not create tmpdir; err=%v", err)
		return
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	type args struct {
		c1, c2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "sources identified as empty path are equal",
			args: args{
				c1: "<empty-path/>",
				c2: "<empty-path/>",
			},
			want: true,
		},
		{
			name: "source identified as empty path are equal to source with empty content",
			args: args{
				c1: "<empty-path/>",
				c2: "",
			},
			want: true,
		},
		{
			name: "empty source are equal",
			args: args{
				c1: "",
				c2: "",
			},
			want: true,
		},
		{
			name: "All line comments or empty lines content considered eq to empty content",
			args: args{
				c1: `// line1
						//lineb
								
						//line 3
						`,
				c2: "",
			},
			want: true,
		},
		{
			name: "All line comment content considered eq to no available",
			args: args{
				c1: "// line1",
				c2: "<not-existing-path/>",
			},
			want: true,
		},
		{
			name: "different single comment sources are equal",
			args: args{
				c1: " // line1",
				c2: "// line1 XXX",
			},
			want: true,
		},

		{
			name: "multiline sources are equals if their actual diff are line comments",
			args: args{
				c1: ` // line1
						lin2
						//l3
						lin4
						`,
				c2: ` // line1
						lin2
						//l3
						lin4
						// comment1
						`,
			},
			want: true,
		},
		{
			name: "multiline sources with different content are not equal",
			args: args{
				c1: ` // line1
						lin2
						//l3
						lin4XXXXXX
						`,
				c2: ` // line1
						lin2
						//l3
						lin4
						// comment1
						`,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src1Path, err := toFileContaining(tmpDir, tt.args.c1)
			if err != nil {
				t.Error(err)
				return
			}
			src2Path, err := toFileContaining(tmpDir, tt.args.c2)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := EqualIgnoreLineCommentPath(src1Path, src2Path)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasLinewiseDiffWhenIgnroingLineCommentsWithPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasLinewiseDiffWhenIgnroingLineCommentsWithPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func toFileContaining(tmpDir string, c2 string) (path string, err error) {
	if c2 == "<not-existing-path/>" {
		path = filepath.Join(tmpDir, "this-file-does-not-exists"+strconv.Itoa(time.Now().Nanosecond()))
		return path, nil
	}
	if c2 == "<empty-path/>" {
		return "", nil
	}
	candidateFile, err := ioutil.TempFile(tmpDir, "candidate-*")
	if err != nil {
		err = errors.Wrapf(err, "fail to create file in [%s]", tmpDir)
		return "", err
	}
	path = candidateFile.Name()
	defer candidateFile.Close()
	if c2 != "" {
		_, err = candidateFile.WriteString(c2)
		if err != nil {
			err = errors.Wrapf(err, "fail to write [%s] into file %s", c2, path)
			return "", err
		}
	}
	return path, nil
}
