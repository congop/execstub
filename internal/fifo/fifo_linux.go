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

// +build !windows

package fifo

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/congop/execstub/internal/rand"
	fifo2 "github.com/containerd/fifo"
)

// OpenFifo opens a fifo. Returns io.ReadWriteCloser.
// Context can be used to cancel this function until open(2) has not returned.
// Accepted flags:
// - syscall.O_CREAT - create new fifo if one doesn't exist
// - syscall.O_RDONLY - open fifo only from reader side
// - syscall.O_WRONLY - open fifo only from writer side
// - syscall.O_RDWR - open fifo from both sides, never block on syscall level
// - syscall.O_NONBLOCK - return io.ReadWriteCloser even if other side of the
//     fifo isn't open. read/write will be connected after the actual fifo is
//     open or after fifo is closed.
func OpenFifo(ctx context.Context, fn string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return fifo2.OpenFifo(ctx, fn, flag, perm)
}

func Mkfifo(path string, mode uint32) (err error) {
	return syscall.Mkfifo(path, mode)
}

func NewFifoNamesForIpc(
	path string,
) (stubberPipePath string, testProcessHelperPipePath string) {
	randStr := rand.NextRandInt63AsHexStr()
	stubberPipePath = path + "_stubber_pipe_" + randStr
	testProcessHelperPipePath = path + "_testprocesshelper_pipe_" + randStr
	return stubberPipePath, testProcessHelperPipePath
}
