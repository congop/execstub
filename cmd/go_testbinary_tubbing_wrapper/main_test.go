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

package main

import (
	"bytes"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/congop/execstub/internal/fifo"
	"github.com/congop/execstub/internal/ipc"
	comproto "github.com/congop/execstub/pkg/comproto"

	"github.com/stretchr/testify/assert"
)

func TestProcHelperAltOutcome(t *testing.T) { //nolint:unparam
	outcome := comproto.ExecOutcome{
		Stderr:   "TestProcHelper1_err",
		Stdout:   "TestProcHelper1_out",
		ExitCode: 2}
	comproto.EffectuateAlternativeExecOutcome(comproto.AdaptOutcomeToCmdStub(&outcome))
}

func TestProcHelperConfiguredOutcome(t *testing.T) { //nolint:unparam
	comproto.EffectuateConfiguredExecOutcome(nil)
}

func Test_runStubbingExec(t *testing.T) {

	type args struct {
		cmdArgs          []string
		createComChannel bool
		cmdConfig        comproto.CmdConfig
	}
	tests := []struct {
		name         string
		args         args
		wantExitCode int
		wantErr      bool
		wantOutcome  comproto.ExecOutcome
	}{
		{
			name: "Should effectuate configured static outcome using test process helper",
			args: args{
				cmdArgs: []string{"arg1", "argb"},
				cmdConfig: comproto.CmdConfig{
					CmdToStub:               "Exe1",
					StubKey:                 "Exe1_9876",
					UnitTestExec:            os.Args[0],
					TestHelperProcessMethod: "TestProcHelperConfiguredOutcome",
					ExitCode:                0,
					TxtStderr:               "configured Err1",
					TxtStdout:               "configured Out1",
					Timeout:                 comproto.CfgDefaultTimeout(),
				},
			},
			wantErr:      false,
			wantExitCode: 0,
			wantOutcome: comproto.ExecOutcome{
				ExitCode:       0,
				Key:            "",
				Stderr:         "configured Err1",
				Stdout:         "configured Out1",
				InternalErrTxt: "",
			},
		},
		{
			name: "Should effectuate alternative outcome using test process helper",
			args: args{
				cmdArgs: []string{"arg1", "argb"},
				cmdConfig: comproto.CmdConfig{
					CmdToStub:               "Exe1",
					StubKey:                 "Exe1_9876",
					UnitTestExec:            os.Args[0],
					TestHelperProcessMethod: "TestProcHelperAltOutcome",
					ExitCode:                0,
					TxtStderr:               "ignored Err1",
					TxtStdout:               "ignore Out1",
					Timeout:                 comproto.CfgDefaultTimeout(),
				},
			},
			wantErr:      false,
			wantExitCode: 0,
			wantOutcome: comproto.ExecOutcome{
				ExitCode:       2,
				Key:            "",
				Stderr:         "TestProcHelper1_err",
				Stdout:         "TestProcHelper1_out",
				InternalErrTxt: "",
			},
		},
		{
			name: "Should effectuate configured static outcome",
			args: args{
				cmdArgs: []string{"arg1", "argb"},
				cmdConfig: comproto.CmdConfig{
					CmdToStub:               "Exe2",
					StubKey:                 "Exe2_9876",
					UnitTestExec:            os.Args[0],
					TestHelperProcessMethod: "",
					ExitCode:                33,
					TxtStderr:               "static-spec-Err1",
					TxtStdout:               "static-spec-Out1",
					Timeout:                 comproto.CfgDefaultTimeout(),
				},
			},
			wantErr:      false,
			wantExitCode: 0,
			wantOutcome: comproto.ExecOutcome{
				ExitCode:       33,
				Key:            "",
				Stderr:         "static-spec-Err1",
				Stdout:         "static-spec-Out1",
				InternalErrTxt: "",
			},
		},

		{
			name: "should effectuate dynamic outcome",
			args: args{
				cmdArgs: []string{"arg1", "argb"},
				cmdConfig: comproto.CmdConfig{
					CmdToStub:               "Exe3",
					StubKey:                 "Exe3_9876",
					UnitTestExec:            os.Args[0],
					TestHelperProcessMethod: "",
					ExitCode:                nil,
					TxtStderr:               "",
					TxtStdout:               "",
					Timeout:                 comproto.CfgDefaultTimeout(),
				},
				createComChannel: true,
			},
			wantErr:      false,
			wantExitCode: 0,
			wantOutcome: comproto.ExecOutcome{
				ExitCode:       255,
				Key:            "",
				Stderr:         "fetch-from-com-channel-Err1",
				Stdout:         "fetch-from-com-channel-Out1",
				InternalErrTxt: "XXX",
			},
		},
		{
			name: "should effectuate dynamic outcome using test process helper",
			args: args{
				cmdArgs: []string{"arg1", "argb"},
				cmdConfig: comproto.CmdConfig{
					CmdToStub:               "Exe3",
					StubKey:                 "Exe3_9876",
					UnitTestExec:            os.Args[0],
					TestHelperProcessMethod: "TestProcHelperConfiguredOutcome",
					ExitCode:                nil,
					TxtStderr:               "",
					TxtStdout:               "",
					Timeout:                 comproto.CfgDefaultTimeout(),
				},
				createComChannel: true,
			},
			wantErr:      false,
			wantExitCode: 0,
			wantOutcome: comproto.ExecOutcome{
				ExitCode:       255,
				Key:            "",
				Stderr:         "fetch-from-com-channel-Err1",
				Stdout:         "fetch-from-com-channel-Out1",
				InternalErrTxt: "XXX",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			stubExecPath, dataDir, comChannel := createTmpDirWithCmdAndConfig(t, tt.args.cmdConfig, tt.args.createComChannel)
			if stubExecPath == "" {
				t.Fatal("Fail to createTmpDirWithCmdAndConfig")
			}
			defer func() {
				if comChannel != nil {
					comChannel.Close()
				}
				os.RemoveAll(filepath.Dir(stubExecPath))
			}()
			originalExecArgs := make([]string, 0, len(tt.args.cmdArgs)+1)
			originalExecArgs = append(originalExecArgs, stubExecPath)
			originalExecArgs = append(originalExecArgs, tt.args.cmdArgs...)
			var gotRequest *comproto.StubRequest
			if comChannel != nil {
				go func() {
					req, ok := <-comChannel.StubRequestChan
					t.Logf("Received (%t) request:%v", ok, req)
					if !ok {
						return
					}
					reqCopy := *req
					gotRequest = &reqCopy
					resp := tt.wantOutcome
					comChannel.ExecResponseChan <- &resp
				}()
			}

			///
			gotExitCode, err := runStubbingExec(stubExecPath, originalExecArgs, &stderr, &stdout)
			///

			if (err != nil) != tt.wantErr {
				t.Errorf("runStubbingExec() error =%s %v, wantErr %v", err.Error(), err, tt.wantErr)
				return
			}
			gotStdErr := stderr.String()
			gotStdOut := stdout.String()
			gotOutcome := comproto.ExecOutcome{
				Key:            "",
				ExitCode:       uint8(gotExitCode),
				Stderr:         gotStdErr,
				Stdout:         gotStdOut,
				InternalErrTxt: "",
			}
			assert.Equal(t, mapToActualOutomeGivenInternalErr(tt.wantOutcome), gotOutcome)
			if tt.args.cmdConfig.UseDynamicOutcome() {
				assert.Equal(
					t,
					&comproto.StubRequest{
						Key:     tt.args.cmdConfig.StubKey,
						Args:    tt.args.cmdArgs,
						CmdName: tt.args.cmdConfig.CmdToStub,
					},
					gotRequest)
			}

			if tt.args.cmdConfig.UseStaticOutCome() {
				repo := comproto.NewStubRequestDirRepo(dataDir)
				gotRequests, err := repo.FindAll()
				assert.NoError(t, err, "fail to find requests")
				assert.Equal(
					t,
					[]comproto.StubRequest{{
						Key:     tt.args.cmdConfig.StubKey,
						Args:    tt.args.cmdArgs,
						CmdName: tt.args.cmdConfig.CmdToStub,
					}},
					*gotRequests)

			}
		})
	}
}

func mapToActualOutomeGivenInternalErr(outcome comproto.ExecOutcome) comproto.ExecOutcome {
	if outcome.InternalErrTxt != "" {
		outcome.Stderr += outcome.InternalErrTxt
		outcome.ExitCode = math.MaxUint8
		outcome.InternalErrTxt = ""
	}
	return outcome
}

func createTmpDirWithCmdAndConfig(
	t *testing.T, testConfig comproto.CmdConfig,
	createComChannel bool,
) (stubExecPath string, dataDir string, comChannel *ipc.StubbingComChannel) {
	var err error

	testDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatalf("Error getting absolute dir for %s:", filepath.Dir(os.Args[0]))
	}
	pattern := testConfig.StubKey + "_*"
	dir, err := ioutil.TempDir(testDir, pattern)
	defer func() {
		// The following cleanup only works if the createTmpDirWithCmdAndConfig
		// local global-ish err is not overshadowed by other err varial redefinition
		//  in subscopes ==> avoid redefinition(mind := with err in left expression)
		if err != nil {
			if comChannel != nil {
				comChannel.Close()
			}
			if dir != "" {
				os.RemoveAll(dir)
			}
		}
	}()
	if err != nil {
		t.Errorf("Could not create temp dir at os tempdir with pattern:%s, err=%v", pattern, err)
		return
	}

	dataDir = filepath.Join(dir, "data")
	err = os.Mkdir(dataDir, 0770)
	if err != nil {
		t.Errorf("Could not create datadir=%s: err=%v", pattern, err)
		return
	}
	testConfig.DataDir = dataDir

	stubExecPath = filepath.Join(dir, testConfig.CmdToStub)

	if createComChannel {
		stubberPipePath, testProcessHelperPipePath := fifo.NewFifoNamesForIpc(stubExecPath)
		comChannel, err = ipc.NewStubbingComChannel(stubberPipePath, testProcessHelperPipePath)
		if err != nil {
			t.Errorf("Error creating com channel: %s %##v", err.Error(), err)
			return
		}
		testConfig.PipeStubber = comChannel.StubberPipePath
		testConfig.PipeTestHelperProcess = comChannel.TestProcessHelperPipePath
	}

	err = testConfig.CreateConfigFile(dir)
	if err != nil {
		t.Errorf(
			"Error executing template: "+
				" \n\ttargetconfig-dir=%s \n\tcfg=%v \n\terr=%s \\tn%##v",
			dir, testConfig, err.Error(), err)
		return
	}
	return stubExecPath, dataDir, comChannel
}
