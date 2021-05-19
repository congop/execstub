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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/pkg/errors"
)

func TestStubRequestDirRepo_Save(t *testing.T) {

	type args struct {
		dataSubDir string
		req        StubRequest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should fail because Data dir does not exist",
			args: args{
				dataSubDir: filepath.Join("notsubdir1", "notsubdir2"),
				req:        StubRequest{Key: "legal_1233", CmdName: "legal", Args: nil},
			},
			wantErr: true,
		},

		{
			name: "Should fail because nil request does not have a legal key",
			args: args{
				dataSubDir: "",
				req:        StubRequest{},
			},
			wantErr: true,
		},

		{
			name: "Should persist stubbing request into data dir",
			args: args{
				dataSubDir: "",
				req: StubRequest{
					CmdName: "mycmd1",
					Args:    []string{"arg0", "arg1", "argc"},
					Key:     "kk_223344",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "00-datadir-save-*")
			if err != nil {
				t.Errorf("fail to create tmp dir, err=%v", err)
				return
			}
			defer os.RemoveAll(tmpDir)

			repo := StubRequestDirRepo{
				dataDir: filepath.Join(tmpDir, tt.args.dataSubDir),
			}
			if err := repo.Save(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf(
					"StubRequestDirRepo.Save() \nerror = %v, \nwantErr %v \nreq=%#v \ndatadir:%s",
					err, tt.wantErr, tt.args.req, tt.args.dataSubDir)
			}
			reqFiles, err := repo.globStubRequestSerFile()
			if err != nil {
				t.Fatalf("fail to get request files with pattern= %v", err)
				return
			}
			if err != nil && len(reqFiles) == 0 {
				t.Fatalf(
					"created request file should have been removed on err:\nerr=%v, \nrequest-files=%v",
					err, reqFiles)
				return
			}
			if tt.args.dataSubDir != "" {
				// nothing was written because dataDir does not exists
				return
			}
			reqs, err := repo.FindAll()
			if err != nil {
				t.Errorf("repo fails to findAll with=%v", err)
				return
			}
			wantReqs := []StubRequest{tt.args.req}
			if tt.wantErr {
				// request not written, so nothing in the repository
				wantReqs = []StubRequest{}
			}
			if !reflect.DeepEqual(wantReqs, *reqs) {
				t.Fatalf("persisted request does not match loaded request:\nwant=%#v \ngot =%#v", wantReqs, *reqs)
			}
		})
	}
}

func TestStubRequestDirRepo_MultSaveFindAllDellAll(t *testing.T) {
	reqs := []StubRequest{
		{
			CmdName: "mycmd1",
			Args:    []string{"arg0", "arg1", "argc"},
			Key:     "kk_223344",
		},
		{
			CmdName: "mycmd1c",
			Args:    []string{"argc0", "argc1", "argcc"},
			Key:     "kkc_223344",
		},
	}

	tmpDir, err := ioutil.TempDir("", "00-datadir-save-find-del-multi*")
	if err != nil {
		t.Errorf("fail to create tmp dir, err=%v", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	repo := StubRequestDirRepo{dataDir: tmpDir}

	///
	for _, req := range reqs {
		if err := repo.Save(req); err != nil {
			t.Errorf(
				"StubRequestDirRepo.Save() \nerror = %v, \nreq=%#v \ndatadir:%s",
				err, req, tmpDir)
			return
		}
	}
	///

	notStubbingReqFiles, err := given3NonStubRequestFilesInDataDir(tmpDir)
	if nil != err {
		t.Error(err)
		return
	}

	///
	gotReqs, err := repo.FindAll()
	///

	if err != nil {
		t.Errorf("repo fails to findAll with=%v", err)
		return
	}
	if !reflect.DeepEqual(reqs, *gotReqs) {
		t.Errorf("persisted request does not match loaded request:\nwant=%v \ngot =%v", reqs, *gotReqs)
		return
	}

	///
	err = repo.DeleteAll()
	///
	if nil != err {
		t.Error("DeleteAll failed:", err)
	}

	remaininfFiles, err := listDir(tmpDir)
	if nil != err {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(notStubbingReqFiles, remaininfFiles) {
		t.Errorf(
			"only non stubbing request file should remain unter dataDir after delete all:"+
				"\nwant=%v \ngot =%v",
			notStubbingReqFiles, remaininfFiles)
	}

	///
	gotReqs, err = repo.FindAll()
	if nil != err {
		t.Error(err)
		return
	}
	if len(*gotReqs) != 0 {
		t.Errorf("not request file in dataDir but FindAll found: %v", gotReqs)
	}
}

func listDir(dir string) (dirFileNames []string, err error) {
	if dir == "" {
		return []string{}, fmt.Errorf("dir must not be empty")
	}
	dirFiles, err := ioutil.ReadDir(dir)
	if nil != err {
		return []string{}, errors.Wrapf(err, "could not read dir:%s", dir)
	}
	if len(dirFiles) == 0 {
		return []string{}, nil
	}

	dirFileNames = make([]string, 0, len(dirFiles))
	for _, dirFile := range dirFiles {
		dirFileNames = append(dirFileNames, filepath.Join(dir, dirFile.Name()))
	}
	return dirFileNames, nil
}

func given3NonStubRequestFilesInDataDir(dir string) (notStubbingReqFiles []string, err error) {
	createdFiles := make([]string, 3)
	for i := 0; i < 3; i++ {
		tmpFile, err := ioutil.TempFile(dir, "believe_me_i_am_really_not_a_req_file_*")
		if nil != err {
			err = errors.Wrapf(err, "fail to create non-stubrequest tmpFile in derectory=%s", dir)
			return []string{}, err
		}
		createdFiles[i] = tmpFile.Name()
	}
	sort.Strings(createdFiles)
	return createdFiles, nil
}
