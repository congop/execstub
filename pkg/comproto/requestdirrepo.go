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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/congop/execstub/internal/rand"

	"sort"

	"github.com/pkg/errors"
)

// StubRequestDirRepo a repository to save and load stubbing requests from a directory.
type StubRequestDirRepo struct {
	dataDir string
}

// NewStubRequestDirRepo creates a new stubbing request repo given the data directory.
func NewStubRequestDirRepo(dataDir string) StubRequestDirRepo {
	return StubRequestDirRepo{dataDir: dataDir}
}

// FindAll return all persisted requests.
func (repo StubRequestDirRepo) FindAll() (requests *[]StubRequest, err error) {
	if err := repo.validateDataDir("FindAll"); err != nil {
		return nil, err
	}
	reqFiles, err := repo.globStubRequestSerFile()
	if err != nil {
		err = errors.Wrap(err, "FindAll could not get request files")
		return nil, err
	}
	lenReqFiles := len(reqFiles)
	if lenReqFiles == 0 {
		return &[]StubRequest{}, nil
	}
	if lenReqFiles > 1 {
		sort.Strings(reqFiles)
	}

	reqs := make([]StubRequest, 0, len(reqFiles))
	for _, reqFile := range reqFiles {
		dstFile, err := os.OpenFile(reqFile, os.O_RDONLY, 0660)
		if err != nil {
			err = errors.Wrapf(err,
				"FindAll could not open rquest file for reading, reqFile=%s",
				reqFile)
			return nil, err
		}
		req, err := StubRequestDecoderFunc(dstFile)
		if err != nil {
			dstFile.Close()
			err = errors.Wrapf(err, "fail to decode request from file:%s", reqFile)
			return nil, err
		}
		dstFile.Close()
		reqs = append(reqs, *req)
	}

	return &reqs, nil
}

// DeleteAll deletes all persisted requests.
func (repo StubRequestDirRepo) DeleteAll() error {
	if err := repo.validateDataDir("FindAll"); err != nil {
		return err
	}
	reqFiles, err := repo.globStubRequestSerFile()
	if err != nil {
		err = errors.Wrapf(err, "DeleteAll could not get request files")
		return err
	}

	bogusRemovals := make([]string, 0, len(reqFiles))
	for _, rf := range reqFiles {
		err := os.Remove(rf)
		if err != nil {
			bogusRemovals = append(bogusRemovals, err.Error())
		}
	}
	if len(bogusRemovals) != 0 {
		return errors.Errorf("DeleteAll could not delete all request file: %v", bogusRemovals)
	}
	return nil
}

// Save saves the given request.
func (repo StubRequestDirRepo) Save(req StubRequest) error {
	if err := assertStubKeyHasRightFormat(req.Key, fmt.Sprintf("req-to-save-to-repo:::%#v", req)); err != nil {
		return err
	}
	if err := repo.validateDataDir("Save"); err != nil {
		return err
	}

	targetFileName := nextStubRequestFileName()
	targetFilePath := filepath.Join(repo.dataDir, targetFileName)
	dstFile, err := os.OpenFile(targetFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		err = errors.Wrapf(err,
			"fail to create file to write stubbing request:%s", targetFilePath)
		return err
	}
	defer func() {
		dstFile.Close()
		if err != nil {
			os.Remove(targetFilePath)
		}
	}()
	encoderFunc := StubRequestEncoderFunc(dstFile)
	err = encoderFunc(&req)
	if nil != err {
		err = errors.Wrapf(err, "faild to encode stuubbing request %#v", &req)
		return err
	}
	return nil
}

func (repo StubRequestDirRepo) validateDataDir(action string) error {
	if repo.dataDir == "" {
		return errors.Errorf("dataDir must not be empty, so cannot do <%s>", action)
	}
	return nil
}

func nextStubRequestFileName() string {
	// ensures names are chronologically ordered in case the cpu is so faster that
	// 1 millisecond time resolution will result into duplicates name
	time.Sleep(2 * time.Millisecond)
	now := time.Now()
	nowStr := now.Format("2006-01-02-15:04:05.000000000")

	nowStr = strings.ReplaceAll(nowStr, ".", "-")
	nowStr = strings.ReplaceAll(nowStr, ":", "-")
	return fmt.Sprintf("ser_stubrequest_%s_019035%0.6d", nowStr, int(rand.NextUint16()))
}

func (repo StubRequestDirRepo) globStubRequestSerFile() (reqFiles []string, err error) {
	reqFilePattern := filepath.Join(repo.dataDir, "ser_stubrequest_*")
	reqFiles, err = filepath.Glob(reqFilePattern)
	if err != nil {
		err = errors.Wrapf(err,
			"could not glob request files with pattern= %s", reqFilePattern)
	}
	return
}
