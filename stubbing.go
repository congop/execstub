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
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/congop/execstub/internal/assets"
	"github.com/congop/execstub/internal/ipc"
	"github.com/congop/execstub/internal/rand"
	comproto "github.com/congop/execstub/pkg/comproto"
	"github.com/pkg/errors"
)

type cmdStubbingSpec struct {
	cmdStub             func(sreq comproto.StubRequest) *comproto.ExecOutcome
	instDirStruct       *stubExecInstallationDirStructure
	comChannel          *ipc.StubbingComChannel
	resetDiscoverySetup func()
}

// ExecStubber provides a mechanism to stub command executions.
type ExecStubber struct {
	cmdStabStore map[string]*cmdStubbingSpec
	mutex        sync.Mutex
}

// CleanUp cleanup resources allocated for stubbing.
// The stubber become unsable.
func (stubber *ExecStubber) CleanUp() {
	stubber.mutex.Lock()
	defer stubber.mutex.Unlock()
	for _, spec := range stubber.cmdStabStore {
		// TODO donot let some panicking prevent attempts to cleanup every spec
		spec.cleanUp()
	}
	stubber.cmdStabStore = nil
}

func (spec *cmdStubbingSpec) cleanUp() {
	if nil != spec.resetDiscoverySetup {
		spec.resetDiscoverySetup()
	}
	if spec.comChannel != nil {
		spec.comChannel.Close()
	}
	os.Remove(spec.instDirStruct.homeDir)
}

// FindAllPersistedStubRequests returns all stub-requests which have
// been persisted into the data basis belonging to the stubbing specification
// identified by the given key.
// Note that stub-requests are only persisted in static mode.
func (stubber *ExecStubber) FindAllPersistedStubRequests(
	stubKey string,
) (peristedRequests *[]comproto.StubRequest, err error) {
	stubber.mutex.Lock()
	spec, ok := stubber.cmdStabStore[stubKey]
	stubber.mutex.Unlock()
	if !ok {
		return &[]comproto.StubRequest{}, nil
	}
	repo := comproto.NewStubRequestDirRepo(spec.instDirStruct.dataDir)
	return repo.FindAll()
}

// DeleteAllPersistedStubRequests deletes all stub-requests which have
// been persisted into the data basis belonging to the stubbing specification
// identified by the given key.
func (stubber *ExecStubber) DeleteAllPersistedStubRequests(
	stubKey string,
) (err error) {
	stubber.mutex.Lock()
	spec, ok := stubber.cmdStabStore[stubKey]
	stubber.mutex.Unlock()
	if !ok {
		return nil
	}
	repo := comproto.NewStubRequestDirRepo(spec.instDirStruct.dataDir)
	return repo.DeleteAll()
}

// NewExecStubber constructs a new ExecStubber.
func NewExecStubber() *ExecStubber {
	return &ExecStubber{cmdStabStore: make(map[string]*cmdStubbingSpec)}
}

// WhenExecDoStubFunc configures stubbing so that the execution of the command
// is replaced by the call of StubFunc.
// The following default are used for settings:
// - Discovery: by PATH
// - Stubbing mode: dynamic
// - Stub exececuatble: Exec based
func (stubber *ExecStubber) WhenExecDoStubFunc(
	cmdStub comproto.StubFunc, cmdToStub string,
	settings comproto.Settings,
) (key string, err error) {
	stubber.mutex.Lock()
	defer stubber.mutex.Unlock()

	cmdToStubStrimmed := strings.TrimSpace(cmdToStub)
	// We cannot concurrently stub an exec with different stubfunction within
	// the same process hierarchy, because we are mutating the env to
	// allow the exec-helper to be discovered instead of the actual executable.
	// Using a combination of command to be stub and process id helps detect
	// this concurrent stubbing.
	// We choose to replace the old stubbing.
	// This should most of the case agree with the normal usage pattern
	// (i.e. sequencially stubbing the same exec without clean-up).
	// Replacing means that we are cleaning up the old stubbing setup.
	// This should lead to failure of test base on parallel stubbing.
	// Which is okay because we do not support parallel stubbing setting.
	// The nee to detect concurrent stubbing setting prohibit the use
	// of the concept while forming the stub key:
	// 	- t.Name (mind sub-test running in parallel in the same process)
	//  - random parts
	stubKey := fmt.Sprint(cmdToStubStrimmed, "_", strconv.Itoa(os.Getpid()))

	// Because we will be teaking environment PATH and executable home we are limited to
	// one stubbing configuration for a given command per test process
	// So we can just replace the stored cmdStub if already in place
	if spec, ok := stubber.cmdStabStore[stubKey]; ok {
		log.Printf("[Warning] discarding old stub setting:%s %v", stubKey, spec)
		stubber.unregisterNotThreadSafe(stubKey)
	}

	// register
	if isPath(cmdToStub) {
		// we will not stub cmd defined by absolute or relatif path
		// because this will involve overriding it or changing current dir
		err = fmt.Errorf(
			"cmdToStub must not be a path (neither absolute, nor relatif): cmdToStub=%s trimmed:=%s",
			cmdToStub, cmdToStubStrimmed)
		return "", err
	}

	instDirStruct, resetDiscoverySetup, err :=
		setupStubbingExecutable(cmdStub, cmdToStubStrimmed, stubKey, settings)
	if err != nil {
		if nil != resetDiscoverySetup {
			resetDiscoverySetup()
		}
		// more clean up left to caller?
		// stubExecutablePath is sub directory of unitest test dir, so it
		// should eventually be clean up by "go-test mechanism"
		return "", err
	}
	spec := cmdStubbingSpec{
		cmdStub:             cmdStub,
		instDirStruct:       instDirStruct,
		resetDiscoverySetup: resetDiscoverySetup,
	}
	stubber.cmdStabStore[stubKey] = &spec
	if settings.InModStatic() {
		// No need for a communication channel for static outcome
		return stubKey, nil
	}
	// TODO setup com channel only for dynamic mode
	comChannel, err := ipc.NewStubbingComChannel(instDirStruct.execPath)
	if err != nil {
		spec.cleanUp()
		return "", err
	}

	spec.comChannel = comChannel

	//  bulkheading by using request/response per exec,
	//  therefore not using 2 central channels for all stubbing
	//  lookup by key of spec is therefore not need to get the comChannels
	go func() {
		for {
			req, ok := <-comChannel.StubRequestChan
			if !ok {
				log.Println("Stop responding to stub requets from StubRequestChan; req=", req)
				return
			}
			log.Printf("Stubbing requested:%#v", req)
			// Not using closure cmdStub because store cmdToStub can be removed
			// Using closure spec directly cannot detect the removal.
			//	- therefore using <central locking> through doStub(..)
			resp := stubber.doStub(*req)
			if resp.InternalErrTxt != "" {
				resp.ExitCode = math.MaxUint8
			}

			comChannel.ExecResponseChan <- resp
			log.Printf("Stubbing ExecOutcome queued resp=%#v req:%#v", resp, req)
		}

	}()

	return stubKey, nil
}

func isPath(testee string) bool {
	return testee != filepath.Base(testee)
}

// Unregister remove the stubbing spec associated with the given key.
func (stubber *ExecStubber) Unregister(key string) {
	stubber.mutex.Lock()
	defer stubber.mutex.Unlock()

	stubber.unregisterNotThreadSafe(key)
}

func (stubber *ExecStubber) unregisterNotThreadSafe(key string) {
	log.Println("Unregistering key=", key)
	spec, ok := stubber.cmdStabStore[key]
	if ok {
		delete(stubber.cmdStabStore, key)
	}

	if spec == nil {
		return
	}

	spec.cleanUp()
}

func (stubber *ExecStubber) getStubbedExecKeys() []string {
	stubber.mutex.Lock()
	defer stubber.mutex.Unlock()
	keys := make([]string, 0, len(stubber.cmdStabStore))
	for key := range stubber.cmdStabStore {
		keys = append(keys, key)
	}
	return keys
}

func (stubber *ExecStubber) doStub(sreq comproto.StubRequest) *comproto.ExecOutcome {

	stubber.mutex.Lock()
	spec, ok := stubber.cmdStabStore[sreq.Key]
	var cmdStub comproto.StubFunc
	if spec != nil {
		cmdStub = spec.cmdStub
	}
	stubber.mutex.Unlock()

	if !ok {
		return &comproto.ExecOutcome{
			Key:      sreq.Key,
			ExitCode: math.MaxUint8,
			InternalErrTxt: fmt.Sprintf(
				"Cannot <DoStub> because cmd has not been registered or has been removed: "+
					"\nrequest=%#v \navailable keys:%s",
				sreq, stubber.getStubbedExecKeys()),
		}
	}
	return cmdStub(sreq)
}

func setupStubbingExecutable(
	cmdStub comproto.StubFunc, cmdToStub string, stubKey string,
	settings comproto.Settings,
) (instDirStruct *stubExecInstallationDirStructure, resetDiscoverySetup func(), err error) {

	testExecutable := os.Args[0]

	instDirStruct, err = createStubExecutableInstDirStucture(stubKey, cmdToStub, settings)
	if nil != err {
		return nil, nil, err
	}

	err = createStubExecConfigFile(
		stubKey, cmdToStub, cmdStub, settings, testExecutable, instDirStruct)
	if nil != err {
		os.RemoveAll(instDirStruct.homeDir)
		return nil, nil, err
	}

	err = createStubExecFile(instDirStruct, settings.ExecType)
	if nil != err {
		os.RemoveAll(instDirStruct.homeDir)
		return nil, nil, err
	}

	resetDiscoverySetup = makeExecStubBeDiscoveredInsteadOfAcualExec(instDirStruct.homeDir, settings)
	log.Printf(
		"Stubbing executable installation structure: \nstub instExecStrubf=%#v \nPATH=%s",
		instDirStruct, os.Getenv("PATH"))

	return instDirStruct, resetDiscoverySetup, nil

}

type stubExecInstallationDirStructure struct {
	homeDir  string
	binDir   string
	execPath string
	dataDir  string
}

func createStubExecutableInstDirStucture(
	stubKey string, cmdToStub string, settings comproto.Settings,
) (instDirStruct *stubExecInstallationDirStructure, err error) {
	testExecutable := os.Args[0]
	unittestExecDir := filepath.Dir(testExecutable)
	// Concurrent stub setting should not be dealing with the
	// same directory. We therefore add a random suffix
	// stubExecDir := filepath.Join(unittestExecDir, stubKey+"_"+ipc.NextRandInt63AsHexStr())
	// stubExecHomeDir := stubExecDir
	homeDir := filepath.Join(unittestExecDir, stubKey+"_"+rand.NextRandInt63AsHexStr())
	dataDir := filepath.Join(homeDir, "data")
	execDir := homeDir
	if comproto.DiscoveredByHomeBinDir == settings.DiscoveredBy {
		if len(settings.DiscoveredByHomeDirBinData.BinDirs) > 0 {
			binDirs := filepath.Join(settings.DiscoveredByHomeDirBinData.BinDirs...)
			execDir = filepath.Join(homeDir, binDirs)
		}
	}
	execPath := filepath.Join(execDir, cmdToStub)
	if err := os.MkdirAll(execDir, 0777); err != nil {
		return nil, errors.Wrapf(err, "fail to create exec dir:%s", execDir)
	}

	if err := os.MkdirAll(dataDir, 0777); err != nil {
		os.RemoveAll(homeDir)
		return nil, errors.Wrapf(err, "fail to create data dir:%s", dataDir)
	}

	dirStucture := stubExecInstallationDirStructure{
		binDir:   execDir,
		dataDir:  dataDir,
		execPath: execPath,
		homeDir:  homeDir,
	}
	return &dirStucture, nil

}

func createStubExecConfigFile(
	stubKey string, cmdToStub string, cmdStub comproto.StubFunc,
	settings comproto.Settings,
	testExecutable string,
	instDirStruct *stubExecInstallationDirStructure,
) error {
	cfg := comproto.CmdConfig{
		CmdToStub:               cmdToStub,
		PipeStubber:             "",
		PipeTestHelperProcess:   "",
		StubKey:                 stubKey,
		TestHelperProcessMethod: settings.TestHelperProcessMethodName,
		Timeout:                 settings.Timeout,
		ExitCode:                nil,
		TxtStderr:               "",
		TxtStdout:               "",
		UnitTestExec:            testExecutable,
		DataDir:                 instDirStruct.dataDir,
	}
	if settings.InModStatic() {
		resp := cmdStub(comproto.StubRequest{})
		cfg.ExitCode = resp.ExitCode
		cfg.TxtStderr = resp.Stderr
		cfg.TxtStdout = resp.Stdout
		if resp.InternalErrTxt != "" {
			cfg.TxtStderr += resp.InternalErrTxt
			cfg.ExitCode = 255
		}
	}
	err := cfg.CreateConfigFile(instDirStruct.binDir)
	if err != nil {
		return errors.Wrapf(err, "fail to write config to %s", instDirStruct.binDir)
	}
	return nil
}

func makeExecStubBeDiscoveredInsteadOfAcualExec(
	stubExecHomeDir string, settings comproto.Settings,
) (resetDiscoverySetup func()) {
	if settings.IsCmdDiscoveredByHomeDir() {
		replacedHome := os.Getenv(settings.DiscoveredByHomeDirBinData.EnvHomeKey)
		os.Setenv(settings.DiscoveredByHomeDirBinData.EnvHomeKey, stubExecHomeDir)
		return toResetDiscoveredByHomeDirFunc(
			settings.DiscoveredByHomeDirBinData.EnvHomeKey,
			stubExecHomeDir, replacedHome)

	}
	envPath := currentEnvPath()
	envPath.positionFirst(stubExecHomeDir)
	os.Setenv("PATH", envPath.String())
	return toResetDiscoveredByPath(stubExecHomeDir)
}

func toResetDiscoveredByHomeDirFunc(
	envHomeKey, stubExecHomeDir, replacedHome string,
) (resetDiscoverySetupFunc func()) {
	return func() {
		// This func provides not provide any mitigation for
		// for comcurrent use of os.Unserenv, os.Getenv and os.SetEnv
		if replacedHome == "" {
			os.Unsetenv(envHomeKey)
		} else {
			currentHomeValue := os.Getenv(envHomeKey)
			// The stubExecHomeDir has a random part so that we are confident
			// that no other stubbing is using the same value.
			// We are therefore resetting to the old value if the current value
			// and our stubExecHomeDir match, so that we donot override
			// setting by another stubbing.
			if currentHomeValue == stubExecHomeDir {
				os.Setenv(envHomeKey, replacedHome)
			}

		}
	}
}

func toResetDiscoveredByPath(stubExecHomeDir string) (resetDiscoverySetupFunc func()) {
	return func() {
		envpath := currentEnvPath()
		// The stubExecHomeDir has a random part so that we are confident
		// that no other stubbing is using the same value.
		// We can therefore remove stubExecHomeDir from the path.
		// Note that this is not a fix for conccurent os.Setenv.
		envpath.removeParts(stubExecHomeDir)
		os.Setenv("PATH", envpath.String())
	}
}

func createStubExecFile(
	instExecInstStruct *stubExecInstallationDirStructure,
	execType comproto.ExecType,
) (err error) {
	// stubExecDir string
	// stubCommandPath = filepath.Join(stubExecDir, cmdToStub)
	stubCommandPath := instExecInstStruct.execPath
	execStubProvider := toExecStubProvider(execType)
	execStubBytes, err := execStubProvider()
	if err != nil {
		return errors.Wrapf(err,
			"fail to get the exec stub from: %T", execStubProvider)
	}
	err = ioutil.WriteFile(stubCommandPath, execStubBytes, 0700) //nolint:gosec
	if nil != err {
		return errors.Wrapf(err,
			"fail to write stub command: err=%v stub cmd path=%s",
			err, stubCommandPath)
	}
	return nil
}

func toExecStubProvider(execType comproto.ExecType) func() (script []byte, err error) {
	if comproto.ExecTypeBash == execType {
		return assets.ExecStubBashScript
	}
	return assets.ExecStubExe
}
