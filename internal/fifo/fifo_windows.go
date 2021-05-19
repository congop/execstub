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

// +build windows

package fifo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/congop/execstub/internal/rand"
	"github.com/pkg/errors"
)

func fileNameWithoutExtTrimSuffix(path string) string {
	fileName := filepath.Base(path)
	ext := filepath.Ext(path)
	return strings.TrimSuffix(fileName, ext)
}

type SimpleFileBaseFifo struct {
	ctx                   context.Context
	baseFn                string
	lastReadFileTimeStamp int64
	lastProcessedFile     string
	lastProccedCount      int
	lastMesg              []byte

	flag int
	perm os.FileMode
}

type message struct {
	msg             []byte
	lastProccedFile string
}

func (fifo *SimpleFileBaseFifo) Read(p []byte) (n int, err error) { // nolint:gocognit

	if fifo.lastMesg != nil {
		// returning part of the buffered message not read yet
		offsettedMsg := fifo.lastMesg[fifo.lastProccedCount:]
		nCopied := copy(p, offsettedMsg)
		if nCopied < len(offsettedMsg) {
			fifo.lastProccedCount += nCopied
		} else {
			fifo.lastProccedCount = 0
			fifo.lastMesg = nil
		}
		return nCopied, nil
	}

	newMesgChan := make(chan *message, 64)
	go func() {

		for {
			// get latest message file
			oldestMessageFile, errGo := fifo.findOldestNotProcessedMessageFile()
			if errGo != nil {
				panic(errGo)
			}
			if oldestMessageFile != "" {
				if _, errGo := os.Stat(oldestMessageFile); errGo == nil {
					msg, errGo := os.ReadFile(oldestMessageFile)
					if errGo == nil {
						os.Remove(oldestMessageFile)
						newMesgChan <- &message{msg: msg, lastProccedFile: oldestMessageFile}
						return
					}
				}
			} else {
				newMesgChan <- &message{msg: []byte{}, lastProccedFile: ""}
			}
			time.Sleep(time.Duration(100 * time.Millisecond))
		}
	}()

	for {
		select {
		case <-fifo.ctx.Done():
			return 0, fifo.ctx.Err()
		case mesg, ok := <-newMesgChan:
			if !ok {
				panic("error reading from newMesgChan")
			}
			if mesg.lastProccedFile == "" {
				// waiting for input
				continue
			} else if mesg.lastProccedFile > fifo.lastProcessedFile {
				offsettedMsg := mesg.msg[fifo.lastProccedCount:]
				nCopied := copy(p, offsettedMsg)
				if nCopied < len(offsettedMsg) {
					// message in file not fully read so we keep it to read the rest
					fifo.lastProccedCount += nCopied
					fifo.lastMesg = mesg.msg
					fifo.lastProcessedFile = mesg.lastProccedFile
				} else {
					fifo.lastProccedCount = 0
					fifo.lastProcessedFile = mesg.lastProccedFile
					fifo.lastMesg = nil
				}

				return nCopied, nil
			}

		}
	}

	// return 0, errors.Errorf("Failed tot read %v", fifo)
}

func (fifo SimpleFileBaseFifo) findOldestNotProcessedMessageFile() (nextToProcess string, err error) {
	pattern := path.Join(fifo.baseFn, "msg_*") // cmdPathPrefix + "*"
	paths, err := filepath.Glob(pattern)
	if err != nil {
		err = errors.Wrapf(err, "fail to Glob bad pattern %s", pattern)
		return "", err
	}

	if len(paths) == 0 {
		return "", nil
	}

	pathsAfterTimestamp := make([]string, 0, len(paths))
	for _, path := range paths {
		if path > fifo.lastProcessedFile {
			pathsAfterTimestamp = append(pathsAfterTimestamp, path)
		}
	}
	// fmt.Printf("\npaths=%v, after=%v, lastProcessed=%s", paths, pathsAfterTimestamp, fifo.lastProcessedFile)
	if len(pathsAfterTimestamp) == 0 {
		return "", nil
	}

	sort.Strings(pathsAfterTimestamp)
	return pathsAfterTimestamp[0], nil
}

func (fifo SimpleFileBaseFifo) Write(p []byte) (n int, err error) {
	// single threaded and at least 1 millisecond resolution, so this will
	// ensure the names are chronologically ordered
	time.Sleep(time.Duration(2 * time.Millisecond))
	nextMessageFile := filepath.Join(fifo.baseFn, fmt.Sprintf("msg_%0.30d", time.Now().UnixNano()))
	err = os.WriteFile(nextMessageFile, p, fifo.perm)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (fifo SimpleFileBaseFifo) Close() error {
	// Nothing to do here because we are communicating by writing and reading file, while not keeping then open.
	// Yes no  cleanup (e.i. deleting message files is done), because this will interact with file messae file being
	// read by the communication peer.
	// Deletion will be eventually done by the test cleanup
	// TODO what about introducing the concept of Destroyer Destroy(..) for finalized cleanup

	return nil
}

func OpenFifoFs(ctx context.Context, fn string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	absFn, err := filepath.Abs(fn)
	if err != nil {
		return nil, errors.Errorf("fail to abs[%s], err=%v", fn, err)
	}
	fifo := SimpleFileBaseFifo{
		ctx:                   ctx,
		baseFn:                absFn,
		lastReadFileTimeStamp: 0,
		lastProcessedFile:     "",
		lastProccedCount:      0,
		flag:                  flag,
		perm:                  perm,
	}
	return &fifo, nil
}

func MkfifoFs(path string, mode uint32) (err error) {
	return nil
}

func NewFifoNamesForFs(
	path string,
) (stubberPipePath string, testProcessHelperPipePath string) {
	filename := fileNameWithoutExtTrimSuffix(path)
	filedir := filepath.Dir(path)
	randStr := rand.NextRandInt63AsHexStr()

	stubberPipePath = filepath.Join(filedir, filename+randStr+"_stubber_pipe_"+"_fifo")
	testProcessHelperPipePath = filepath.Join(filedir, filename+randStr+"_testprocesshelper_pipe_"+"_fifo")
	_ = os.Mkdir(stubberPipePath, 0770)
	_ = os.Mkdir(testProcessHelperPipePath, 0770)
	return stubberPipePath, testProcessHelperPipePath
}

func NewFifoNamesForIpc(
	path string,
) (stubberPipePath string, testProcessHelperPipePath string) {
	return NewFifoNamesForFs(path)
}

func Mkfifo(path string, mode uint32) (err error) {
	return MkfifoFs(path, mode)
}

func OpenFifo(ctx context.Context, fn string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return OpenFifoFs(ctx, fn, flag, perm)
}
