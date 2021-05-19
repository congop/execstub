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
	"os"
	"testing"
)

func TestNewFifoNamesForIpc(t *testing.T) {
	testExecPath := os.Args[0]
	// timeout := time.Duration(10 * time.Second)
	stubberPipePath, testProcessHelperPipePath := NewFifoNamesForIpc(testExecPath)
	if err := Mkfifo(stubberPipePath, 0777); err != nil {
		t.Fatal(err)
	}

	if err := Mkfifo(testProcessHelperPipePath, 0777); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	writer, err := OpenFifo(ctx, stubberPipePath, os.O_WRONLY, 0777)
	if err != nil {
		t.Fatal(err)
	}
	expectedMesg1 := "abcdefghij1234567"
	expectedMesg2 := "ABCDEFGHIJ1234567"
	_, _ = writer.Write([]byte(expectedMesg1))
	_, _ = writer.Write([]byte(expectedMesg2))

	reader, err := OpenFifo(ctx, stubberPipePath, os.O_RDONLY, 0777)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 10)
	mesg1 := make([]byte, 0, 20)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	mesg1 = append(mesg1, buf[:n]...)
	n, err = reader.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	mesg1 = append(mesg1, buf[:n]...)
	mesg1Str := string(mesg1)
	if mesg1Str != expectedMesg1 {
		t.Fatalf("mesg1=%s expected=%s", mesg1Str, expectedMesg1)
	}

	buf = make([]byte, 20)
	n, err = reader.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	mesg2 := make([]byte, 0, 20)
	mesg2 = append(mesg2, buf[:n]...)
	mesg2Str := string(mesg2)
	if mesg2Str != expectedMesg2 {
		t.Fatalf("mesg2=%s expected=%s\n", mesg2Str, expectedMesg2)
	}
}
