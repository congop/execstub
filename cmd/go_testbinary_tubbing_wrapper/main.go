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
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	comproto "github.com/congop/execstub/pkg/comproto"
)

// main for the executable which can be used as stub.
func main() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("error getting executable: %v", err)
		os.Exit(math.MaxInt8)
	}
	exitCode, err := runStubbingExec(execPath, os.Args, os.Stderr, os.Stdout)
	if err != nil {
		fmt.Printf("Error executing stubbing: %v", err)
		os.Exit(math.MaxInt8)
	}
	os.Exit(int(exitCode))
}

func runStubbingExec(
	execPath string, originalExecArgs []string,
	stderr io.Writer, stdout io.Writer,
) (exitCode uint8, err error) {

	cfg, err := comproto.CmdConfigForCommand(execPath)
	if err != nil {
		exitCode = math.MaxUint8
		err = errors.Wrapf(err,
			"Error reading cmd config: \ncmd=%s cmd-path=%s \nenv-path=%s\n",
			originalExecArgs[0], execPath, os.Getenv("PATH"))
		return
	}

	if cfg.UseTestHelperProcess() {
		return outcomeFromTestHelperProcess(
			execPath+".config", cfg, originalExecArgs, stderr, stdout)
	}

	stubReq := cfg.StubRequestWith(originalExecArgs[1:])

	if cfg.UseStaticOutCome() {
		return outcomeFromStaticSpec(stubReq, cfg, stderr, stdout)
	}

	timeout, err := cfg.TimeoutAsDuration()
	if err != nil {
		exitCode = math.MaxUint8
		err = errors.Wrapf(err, "could not get configured timeout")
		return exitCode, err
	}
	exitCodeUint8 := comproto.EffectuateDynamicOutcome(
		timeout, cfg.PipeStubber, cfg.PipeTestHelperProcess,
		stubReq, stderr, stdout)

	return exitCodeUint8, nil

}

func outcomeFromStaticSpec(
	req comproto.StubRequest,
	cfg *comproto.CmdConfig,
	stderr io.Writer, stdout io.Writer,
) (exitCode uint8, err error) {
	repo := comproto.NewStubRequestDirRepo(cfg.DataDir)
	err = repo.Save(req)
	if nil != err {
		err = errors.Wrapf(err, "fail to save err into %s", cfg.DataDir)
		return math.MaxUint8, err
	}
	exitCode, err = cfg.ExitCodeUint8()
	if nil != err {
		return math.MaxUint8, err
	}
	_, err = fmt.Fprint(stderr, cfg.TxtStderr)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Could not write the following to stderr writer(%##v): %s",
			stderr, cfg.TxtStderr)
		return
	}

	_, err = fmt.Fprint(stdout, cfg.TxtStdout)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Could not write the following to stderr writer(%##v): %s",
			stdout, cfg.TxtStdout)
		return
	}
	return exitCode, nil
}

func outcomeFromTestHelperProcess(
	stubCmdConfigPath string, cfg *comproto.CmdConfig, originalExecArgs []string,
	stderr io.Writer, stdout io.Writer,
) (exitCode uint8, err error) {
	helperArgs := make([]string, 0, len(originalExecArgs)+3)
	// # -test.run takes a regex therefore matching the exact test helper process method
	helperArgs = append(helperArgs, "-test.run=^"+cfg.TestHelperProcessMethod+"$", "--")
	helperArgs = append(helperArgs, originalExecArgs[1:]...)

	cmd := exec.Command(cfg.UnitTestExec, helperArgs...) // #nosec
	stubingEnv := append(make([]string, 0, len(cmd.Env)+5),
		"__EXECSTUBBING_GO_WANT_HELPER_PROCESS=1",
		"__EXECSTUBBING_STUB_CMD_CONFIG="+stubCmdConfigPath,
	)
	stubingEnv = append(stubingEnv, cmd.Env...)
	cmd.Env = stubingEnv
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err = cmd.Run()
	if err != nil {

		if _, ok := err.(*exec.ExitError); ok {
			//Cmd run but exited with non zero exit code
			// We are ok with that, as it is the actual command execution outcome
			// the command parent should merely return the child process exit code
			return uint8(cmd.ProcessState.ExitCode()), nil
		}
		// Now this is not an <actual> outcome of the command execution
		// e.g. ErrNotFound.
		// It is some kind internal error.
		// Signaling this with exit code MaxUint8 and error txt in stderr
		err = errors.Wrapf(
			err,
			"Error executing command[%s]: \n\tenv=%s \n\tpath=%s",
			cmd.String(), cmd.Env, cmd.Path,
		)
		exitCode = math.MaxUint8
		return
	}

	return uint8(cmd.ProcessState.ExitCode()), nil
}
