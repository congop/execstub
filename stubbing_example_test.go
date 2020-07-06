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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/congop/execstub/pkg/comproto"
)

func Example_dynamicDefaultSettings() {
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	staticOutcome := comproto.ExecOutcome{
		Stderr:   "err1",
		Stdout:   "sout1",
		ExitCode: 0,
	}
	recStubFunc, reqStore := comproto.RecordingExecutions(
		comproto.AdaptOutcomeToCmdStub(&staticOutcome))
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, "SuperExe", comproto.Settings{})
	ifNotEqPanic(nil, err, "fail to setip stub")

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	//accessing and checking stubrequest dynymic mode
	gotRequests := *reqStore
	wanRequets := []comproto.StubRequest{
		{
			CmdName: "SuperExe", Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, gotRequests, "unexpected stub requests")

	//accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

//used as test process help Settings{TestHelperProcessMethodName: "TestHelperProcExample_dynamic"}
func TestHelperProcExample_dynamic(t *testing.T) {
	comproto.EffectuateConfiguredExecOutcome(nil)
}

func Example_dynamicWithTestHelperProc() {
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	staticOutcome := comproto.ExecOutcome{
		Stderr: "",
		Stdout: `REPOSITORY:TAG
						golang:1.14
						golang:latest
						golang:1.14-alpine3.12
						ubuntu:18.04`,
		ExitCode: 0,
	}
	recStubFunc, reqStore := comproto.RecordingExecutions(
		comproto.AdaptOutcomeToCmdStub(&staticOutcome))
	setting := comproto.Settings{TestHelperProcessMethodName: "TestHelperProcExample_dynamic"}
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, "docker", setting)
	ifNotEqPanic(nil, err, "fail to setip stub")

	//args := []string{"image", "ls", "--format", "\"{{.Repository}}:{{.Tag}}\""}
	args := []string{"image", "ls", "--format", "table '{{.Repository}}:{{.Tag}}'"}
	cmd := exec.Command("docker", args...)
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "exit code set to 0 ==> execution should succeed")

	//accessing and checking stubrequest dynymic mode
	gotRequests := *reqStore
	wanRequets := []comproto.StubRequest{
		{
			CmdName: "docker", Args: args, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, gotRequests, "unexpected stub requests")

	//accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

func Example_static() {
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	staticOutcome := comproto.ExecOutcome{
		Stderr:   "err1",
		Stdout:   "sout1",
		ExitCode: 0,
	}
	recStubFunc, reqStore := comproto.RecordingExecutions(
		comproto.AdaptOutcomeToCmdStub(&staticOutcome))
	settings := comproto.Settings{Mode: comproto.StubbingModeStatic}
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, "SuperExe", settings)
	ifNotEqPanic(nil, err, "fail to setip stub")
	ifNotEqPanic(
		[]comproto.StubRequest{{}}, *reqStore,
		"Static mod evaluate StubFunc at setup with nil arg")
	*reqStore = (*reqStore)[:0]
	//recStubFunc(comproto.StubRequest{Key: "xxx"})

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer
	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	//accessing and checking stubrequest static mode
	ifNotEqPanic(0, len(*reqStore), "Unexpected StubFunc call in static mode")
	gotRequests, err := stubber.FindAllPersistedStubRequests(key)
	ifNotEqPanic(nil, err, "fail to find all persisted stub request")
	wanRequets := []comproto.StubRequest{
		{
			CmdName: "SuperExe", Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, *gotRequests, "unexpected stub requests")

	//accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

//TestHelperProcExample_static is used as test process help .
// Configured wihh: Settings{TestHelperProcessMethodName: "TestHelperProcExample_static"}
func TestHelperProcExample_static(t *testing.T) {
	extraJobOnStubRequest := func(req comproto.StubRequest) error {
		//some extrat side effect
		//we are adding to stdout
		fmt.Print("extra_side_effect_")
		return nil
	}
	comproto.EffectuateConfiguredExecOutcome(extraJobOnStubRequest)
}

func Example_staticWithTestHelperProc() {
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	staticOutcome := comproto.ExecOutcome{
		Stderr:   "err1",
		Stdout:   "sout1",
		ExitCode: 0,
	}
	recStubFunc, reqStore := comproto.RecordingExecutions(
		comproto.AdaptOutcomeToCmdStub(&staticOutcome))
	settings := comproto.Settings{
		Mode:                        comproto.StubbingModeStatic,
		TestHelperProcessMethodName: "TestHelperProcExample_static",
	}
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, "SuperExe", settings)
	ifNotEqPanic(nil, err, "fail to setip stub")
	ifNotEqPanic(
		[]comproto.StubRequest{{}}, *reqStore,
		"Static mod evaluate StubFunc at setup with nil arg")
	*reqStore = (*reqStore)[:0]

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer
	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	//accessing and checking stubrequest static mode
	ifNotEqPanic(0, len(*reqStore), "Unexpected StubFunc call in static mode")
	gotRequests, err := stubber.FindAllPersistedStubRequests(key)
	ifNotEqPanic(nil, err, "fail to find all persisted stub request")
	wanRequets := []comproto.StubRequest{
		{
			CmdName: "SuperExe", Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, *gotRequests, "unexpected stub requests")

	//accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic("extra_side_effect_"+staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

func Example_homeBinDir() {
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	staticOutcome := comproto.ExecOutcome{
		Stderr:   "",
		Stdout:   "%s openjdk version \"11.x.x\" 2020-mm-dd",
		ExitCode: 0,
	}
	recStubFunc, reqStore := comproto.RecordingExecutions(
		comproto.AdaptOutcomeToCmdStub(&staticOutcome))
	settings := comproto.Settings{
		DiscoveredBy: comproto.DiscoveredByHomeBinDir,
		DiscoveredByHomeDirBinData: comproto.DiscoveredByHomeDirBinData{
			EnvHomeKey: "JAVA_HOME",
			BinDirs:    []string{"bin"},
		},
		ExecType: comproto.ExecTypeBash,
	}
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, "java", settings)
	ifNotEqPanic(nil, err, "fail to setip stub")

	javaCmd := os.ExpandEnv("${JAVA_HOME}/bin/java")
	cmd := exec.Command(javaCmd, "-version")
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	//accessing and checking stubrequest dynymic mode

	wantRequests := []comproto.StubRequest{
		{
			CmdName: "java", Args: []string{"-version"}, Key: key,
		},
	}
	ifNotEqPanic(wantRequests, *reqStore, "unexpected stub requests")

	//accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr //stdout:"+bufStdout.String())

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

func ifNotEqPanic(want interface{}, got interface{}, mesg string) {

	if reflect.DeepEqual(want, got) {
		return
	}
	if mesg == "" {
		mesg = "got != want"
	}
	pmesg := fmt.Sprintf(
		"%s: \n\twant=%#v \n\tgot =%#v \n\tgot=%s",
		mesg, want, got, got)
	panic(pmesg)
}
