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
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	rt "github.com/congop/execstub/internal/runtime"
	comproto "github.com/congop/execstub/pkg/comproto"
)

func unStubbableExecCommandExecution(
	t *testing.T, setting *comproto.Settings, cmdName string, cmdArgs ...string,
) *comproto.ExecOutcome {
	cmdNameActual := cmdName
	if setting.IsCmdDiscoveredByHomeDir() {
		homeDirPath := os.Getenv(setting.DiscoveredByHomeDirBinData.EnvHomeKey)
		binSubDir := homeDirPath
		if len(setting.DiscoveredByHomeDirBinData.BinDirs) > 0 {
			binSubDir = filepath.Join(
				binSubDir, filepath.Join(setting.DiscoveredByHomeDirBinData.BinDirs...))
		}
		cmdNameActual = filepath.Join(binSubDir, cmdNameActual)
	}
	cmd := exec.Command(cmdNameActual, cmdArgs...)
	var bout bytes.Buffer
	var berr bytes.Buffer
	cmd.Stdout = &bout
	cmd.Stderr = &berr
	err := cmd.Run()
	internalErrTxt := ""
	if err != nil {
		t.Logf("Exec Error: %##v ===> %s", err, err)
		internalErrTxt = err.Error()
	}
	return &comproto.ExecOutcome{
		Key: "", ExitCode: uint8(cmd.ProcessState.ExitCode()),
		Stderr: berr.String(), Stdout: bout.String(),
		InternalErrTxt: internalErrTxt}
}

func TestHelperProcess2(t *testing.T) {
	if !comproto.StubbingOngoing() {
		return
	}
	os.Exit(66)
}

func TestHelperProcess(t *testing.T) {
	comproto.EffectuateConfiguredExecOutcome(nil)
}

func TestHelperProcessStatic(t *testing.T) {
	comproto.EffectuateConfiguredExecOutcome(nil)
}

func Test_stubExecutableWith(t *testing.T) { // nolint:gocognit

	var execStubber = ExecStubber{cmdStabStore: make(map[string]*cmdStubbingSpec)}
	t.Cleanup(execStubber.CleanUp)
	settings := &comproto.Settings{}

	type args struct {
		cmdToStub   string
		cmdArgs     [][]string
		cmdStub     func(sreq comproto.StubRequest) *comproto.ExecOutcome
		wantError   bool
		wantOutcome comproto.ExecOutcome
	}
	tests := []struct {
		name string
		args args
	}{
		// using -- in --argsb to make sure arguments are not interpreted in the stub in subprocess
		// using %s in <ErrOk%s> to make sur the argument are not falsely use in string formatting
		{
			name: "Exec outcome must be the return of stub commd",
			args: args{
				cmdArgs:   [][]string{{"args1", "--argsb"}, nil},
				cmdToStub: rt.EnsureHasExecExt("BlaBlaExe"),
				cmdStub: func(sreq comproto.StubRequest) *comproto.ExecOutcome {
					t.Logf("Stubbing CMD: %v", sreq)
					// return "OutOk", "ErrOk", 0, nil
					return &comproto.ExecOutcome{
						Key: sreq.Key, ExitCode: 0, Stderr: "ErrOk%s", Stdout: "OutOk%s",
						InternalErrTxt: "",
					}
				},

				wantError: false,
				wantOutcome: comproto.ExecOutcome{
					Key: "", ExitCode: 0,
					Stderr: "ErrOk%s", Stdout: "OutOk%s",
					InternalErrTxt: ""},
			},
		},
		{
			name: "Internal stubbing error should be mapped to non-successful exit code and err-txt send to std-err",
			args: args{
				cmdArgs:   [][]string{{}, {"a1", "b1", "c1"}},
				cmdToStub: rt.EnsureHasExecExt("BlaBlaExe"),
				cmdStub: func(sreq comproto.StubRequest) *comproto.ExecOutcome {
					t.Logf("Stubbing CMD: %v", sreq)
					return &comproto.ExecOutcome{
						Key: sreq.Key, ExitCode: 0, Stderr: "EEE_", Stdout: "OOO_",
						InternalErrTxt: "Err42%s",
					}
				},

				wantError: false,
				wantOutcome: comproto.ExecOutcome{
					Key: "", ExitCode: math.MaxUint8,
					Stderr: "EEE_Err42%s", Stdout: "OOO_",
					InternalErrTxt: "exit status 255"},
			},
		},
	}
	setDiscoveredByFuncs := []func(){
		settings.DiscoveredByPath,
		func() {
			settings.DiscoveredByHomeDirBin("MY_HOME_KEY", "bin", "i386")
		},
	}

	setModeFuncs := []func(){
		func() {
			settings.ModeStatic()
			settings.WithTestProcessHelper("TestHelperProcessStatic")
		},
		func() {
			settings.ModeStatic()
			settings.WithoutTestProcessHelper()
		},
		func() {
			settings.ModeDanymic()
			settings.WithTestProcessHelper("TestHelperProcess")
		},
		func() {
			settings.ModeDanymic()
			settings.WithoutTestProcessHelper()
		},
	}

	// test discovery settings must be passed to unStubbableExecCommandExecution
	// alreadyRun := false
	for _, setDiscoveredByFunc := range setDiscoveredByFuncs {
		setDiscoveredByFunc()

		for _, setExecTypeFunc := range []func(){settings.ExecTypeBash, settings.ExecTypeExe} {
			setExecTypeFunc()
			if settings.IsUsingExecTypeBash() && rt.IsWindows() {
				continue
			}
			for _, setModeFunc := range setModeFuncs {
				setModeFunc()
				for _, tt := range tests {
					t.Run(tt.name+testNamePart(*settings), func(t *testing.T) {
						cmdStub, callArgs := comproto.RecordingExecutions(tt.args.cmdStub)

						key, err := execStubber.WhenExecDoStubFunc(
							cmdStub, tt.args.cmdToStub, *settings)

						if nil != err {
							t.Fatalf("Error resgistering cmd-stub: err=%#v key=%s", err, key)
						}
						defer execStubber.Unregister(key)

						///
						// Make sure code can handle repeated execution
						unStubbableExecCommandExecution(
							t, settings, tt.args.cmdToStub, tt.args.cmdArgs[0]...)
						outcome := unStubbableExecCommandExecution(
							t, settings, tt.args.cmdToStub, tt.args.cmdArgs[1]...)
						///

						assert.Equal(t, tt.args.wantOutcome, *outcome)
						repeatedCallArgs := tt.args.cmdArgs
						expectedReqs := toExpectedRequets(key, tt.args.cmdToStub, repeatedCallArgs)
						if settings.InModDyna() {
							assert.Equal(t, expectedReqs, *callArgs)
						}
						if settings.InModStatic() {
							gotArgs, err := execStubber.FindAllPersistedStubRequests(key)
							assert.NoError(t, err, "fail to find all")
							assert.Equal(t, expectedReqs, *gotArgs)
						}
					})
				}
			}
		}
	}
}

func toExpectedRequets(
	stubKey string, cmdName string, callArgs [][]string,
) []comproto.StubRequest {
	reqs := make([]comproto.StubRequest, 0, len(callArgs))
	for _, args := range callArgs {
		req := comproto.StubRequest{
			CmdName: cmdName,
			// Args:    args,
			Key: stubKey,
		}
		// to follow how deserialization does it,
		// we let req.Args be []string(nil)} (unassigned) instead of []string{}
		if len(args) > 0 {
			req.Args = args
		}
		reqs = append(reqs, req)
	}
	return reqs
}

func testNamePart(s comproto.Settings) string {
	return "_Settings_" + string(s.DiscoveredBy) +
		"_" + string(s.ExecType) + "_" + string(s.Mode) +
		"_" + s.TestHelperProcessMethodName
}

func Test_NoReEntrantLockingSoThatStubberCanBeReused(t *testing.T) {
	// one stubber for multiple sub-tests so that we can test re-usability
	stubber := NewExecStubber()
	defer stubber.CleanUp()
	settings := comproto.SettingsDynaStubCmdDiscoveredByPath()

	type args struct {
		stubFunc comproto.StubFunc
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Should pickup the status done and return error free",
			args: args{
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: done", InternalErrTxt: "",
					},
				}, true),
			},
		},
		{
			name: "Should pickup static error and return with error",
			args: args{
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: error", InternalErrTxt: "",
					},
				}, true),
			},
		},
		{
			name: "Should timeout",
			args: args{
				stubFunc: comproto.AdaptOutcomesToCmdStub([]*comproto.ExecOutcome{
					{
						Key: "", ExitCode: 0, Stderr: "", Stdout: "Value: status: running", InternalErrTxt: "",
					},
				}, true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// settings := comproto.SettingsDynaStubCmdDiscoveredByPath()

			stubFunc, reqs := comproto.RecordingExecutions(tt.args.stubFunc)
			_, err := stubber.WhenExecDoStubFunc(stubFunc, rt.EnsureHasExecExt("MySuperCmd"), *settings)
			if err != nil {
				t.Errorf("Failed  to mock cmd MySuperCmd: %v", err)
				return
			}

			// /////////
			cmd := exec.Command("MySuperCmd")
			actualBytes, err := cmd.Output()
			expected := tt.args.stubFunc(comproto.StubRequest{}).Stdout
			actual := string(actualBytes)
			if (err != nil) || expected != actual {
				t.Errorf(
					"unexpected cmd execution outcome "+
						"\n\terror=%v \n\tcmd=%v \n\treqs=%#v \n\texpected=%s \n\tactual=%s \n\tsuccess=%d",
					err, cmd, reqs, expected, actual, cmd.ProcessState.ExitCode(),
				)
			}

		})
	}
}
