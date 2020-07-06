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
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	ini "gopkg.in/ini.v1"
)

// CmdConfig holds configuration data for an executable used for stubbing
type CmdConfig struct {
	StubKey                 string
	CmdToStub               string
	UnitTestExec            string
	TestHelperProcessMethod string
	TxtStdout               string
	TxtStderr               string
	ExitCode                OptionalUint8
	PipeStubber             string
	PipeTestHelperProcess   string
	Timeout                 OptionalDuration
	DataDir                 string
}

// ExitCodeUint8 returns the configured exit code.
func (cfg CmdConfig) ExitCodeUint8() (exitcode uint8, err error) {
	if nil == cfg.ExitCode {
		return math.MaxUint8, errors.New("exitcode not configured")
	}

	return ValueUint8(cfg.ExitCode)

}

// ExitCodeTxt return the text representation of the exit code.
func (cfg CmdConfig) ExitCodeTxt() string {
	if nil == cfg.ExitCode {
		return ""
	}
	code, err := cfg.ExitCodeUint8()
	if err != nil {
		return fmt.Sprintf("ERROR_%s_%#v", err.Error(), cfg.ExitCode)
	}
	return strconv.Itoa(int(code))
}

// CmdConfigForCommand loads the configuration for the given executable identified by its path.
func CmdConfigForCommand(cmdPath string) (*CmdConfig, error) {
	cmdConfigPath := cmdPath + ".config"
	return CmdConfigLoadedFromFile(cmdConfigPath)
}

// CmdConfigLoadedFromFile loads the configuration from the given file path.
func CmdConfigLoadedFromFile(cmdConfigPath string) (*CmdConfig, error) {
	cmdConfigPathAbs, err := filepath.Abs(cmdConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting absulute path for: %s", cmdConfigPath)
	}
	if !fileExists(cmdConfigPathAbs) {
		return nil, errors.Errorf("config file not found at:%s", cmdConfigPathAbs)
	}

	cfg, err := ini.Load(cmdConfigPathAbs)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading init file: %s", cmdConfigPathAbs)
	}

	cmdCfg := CmdConfig{}

	section := cfg.Section("")
	cmdCfg.StubKey = section.Key("__EXECSTUBBING_STUB_KEY").String()
	cmdCfg.CmdToStub = section.Key("__EXECSTUBBING_CMD_TO_STUB").String()
	cmdCfg.UnitTestExec = section.Key("__EXECSTUBBING_UNIT_TEST_EXEC").String()
	cmdCfg.TestHelperProcessMethod = section.Key("__EXECSTUBBING_TEST_HELPER_PROCESS_METHOD").String()
	cmdCfg.DataDir = section.Key("__EXECSTUBBING_DATA_DIR").String()

	cmdCfg.TxtStdout, err = decodeBase64SectionValue(section, "__EXECSTUBBING_STD_OUT")
	if nil != err {
		return nil, err
	}
	cmdCfg.TxtStderr, err = decodeBase64SectionValue(section, "__EXECSTUBBING_STD_ERR")
	if nil != err {
		return nil, err
	}

	cmdCfg.ExitCode, err = readExitCodeAsUint8(section)
	if err != nil {
		return nil, err
	}

	cmdPathAbs := strings.TrimSuffix(cmdConfigPathAbs, ".config")

	cmdCfg.PipeStubber, err = findLatestFileWithPrefix(cmdPathAbs + "_stubber_pipe_")
	if err != nil {
		return nil, err
	}
	cmdCfg.PipeTestHelperProcess, err = findLatestFileWithPrefix(cmdPathAbs + "_testprocesshelper_pipe_")
	if err != nil {
		return nil, err
	}

	cmdCfg.Timeout, err = timeoutFrom(section)
	if err != nil {
		return nil, err
	}

	return &cmdCfg, nil
}

// CfgDefaultTimeout return default value of timeout in not provided.
func CfgDefaultTimeout() time.Duration {
	return time.Duration(10 * time.Second)
}

func timeoutFrom(section *ini.Section) (timeout time.Duration, err error) {
	timeoutTxt := section.Key("__EXECSTUBBING_TIMEOUT_NANOS").String()
	timeoutTxt = strings.TrimSpace(timeoutTxt)
	if timeoutTxt != "" {
		return CfgDefaultTimeout(), nil
	}
	timeoutInt64, err := strconv.ParseInt(timeoutTxt, 10, 64)
	if nil != err {
		err = errors.Wrapf(err, "could not parse %s as int64", timeoutTxt)
		return CfgDefaultTimeout(), err
	}
	if timeoutInt64 <= 0 {
		err = errors.Errorf("timeout must greater 0 but was %d", timeoutInt64)
		return CfgDefaultTimeout(), err
	}

	return time.Duration(timeoutInt64), nil

}

func decodeBase64SectionValue(section *ini.Section, key string) (decoded string, err error) {
	strBase64 := section.Key(key).String()
	if strBase64 == "" {
		return "", nil
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(strBase64)
	if nil != err {
		return "", errors.Wrapf(err, "fail to base64 decode sectin[%s]=%s", key, strBase64)
	}
	return string(decodedBytes), nil
}

func findLatestFileWithPrefix(cmdPathPrefix string) (latestPathHavingThisPrefix string, err error) {
	pattern := cmdPathPrefix + "*"
	paths, err := filepath.Glob(pattern)
	if err != nil {
		err = errors.Wrapf(err, "fail to Glob bad pattern %s", pattern)
		return "", err
	}
	switch len(paths) {
	case 1:
		return paths[0], nil
	case 0:
		return "", nil
	default:
		path := ""
		latestTime := time.Unix(0, 0)
		for _, c := range paths {
			stat, err := os.Stat(c)
			if err != nil {
				//some how we cannot access this file information
				//possible cause: access denied
				// what ever the cause, this file does not exists for us
				// therefore not considering it
				// not print out to avoid polluting std-out/err and mixing with actual stubbing outcome
				continue
			}
			ft := stat.ModTime()
			if ft.After(latestTime) {
				path = c
			}
		}
		return path, nil
	}
}

// UseTestHelperProcess tells whether the use of test helper process is configured.
func (cfg CmdConfig) UseTestHelperProcess() bool {
	return cfg.TestHelperProcessMethod != ""
}

// UseDynamicOutcome tells whether the use of dynamic outcome is configured.
func (cfg CmdConfig) UseDynamicOutcome() bool {
	//return "" != cfg.PipeStubber || "" != cfg.PipeTestHelperProcess
	return !cfg.UseStaticOutCome()
}

// UseStaticOutCome stells whether the us of static outcome is configured.
func (cfg CmdConfig) UseStaticOutCome() bool {
	return cfg.TxtStderr != "" || cfg.TxtStdout != "" || cfg.ExitCode != nil
}

// StderrAvail return true if stderr data are available; false otherwise.
func (cfg CmdConfig) StderrAvail() bool {
	return cfg.TxtStderr != ""
}

// StdoutAvail return true if stdout data are available; false otherwise.
func (cfg CmdConfig) StdoutAvail() bool {
	return cfg.TxtStdout != ""
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// CreateConfigFile create a file holding the configuration state at the given directory.
func (cfg CmdConfig) CreateConfigFile(
	cmdDir string,
) error {
	var err error
	var tmpl *template.Template

	tmplBytes, err := CmdConfigTmpl()
	if err != nil {
		return errors.Wrap(err, "error getting CmdConfigTmpl")
	}
	tmpl = template.New("stub-cmd-config")
	tmpl = tmpl.Funcs(template.FuncMap{"base64": toBase64})
	tmpl, err = tmpl.Parse(string(tmplBytes))
	if err != nil {
		return errors.Wrapf(err, "error parsing bytes\n %s", string(tmplBytes))
	}

	configFilePath := filepath.Join(cmdDir, cfg.CmdToStub+".config")
	configFile, err := os.Create(configFilePath)
	if err != nil {
		return errors.Wrapf(err, "could not creat config file %s", configFilePath)
	}
	err = tmpl.Execute(configFile, cfg)

	if err != nil {
		return errors.Wrapf(err,
			"error executing template: \n\ttargetconfig-path=%s ", configFilePath)
	}

	return nil
}

func toBase64(str string) (strBase64 string) {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func readExitCodeAsUint8(section *ini.Section) (exitCode OptionalUint8, err error) {
	txt := section.Key("__EXECSTUBBING_EXIT_CODE").String()
	if txt == "" {
		return nil, nil
	}

	code, err := strconv.ParseUint(txt, 10, 64)
	if err != nil {
		return uint8(math.MaxUint8), errors.Wrapf(err, "could not parse exit code as uint8:%s ", txt)
	}

	if code > math.MaxUint8 {
		err = errors.Errorf(
			"exit code must not be greater %d but was %d from text=%s",
			math.MaxUint8, code, txt)

		return uint8(math.MaxUint8), err

	}

	return uint8(code), nil
}

// TimeoutAsFormattedNanos return a string nanos integer representation of the  configured timeout.
func (cfg CmdConfig) TimeoutAsFormattedNanos() string {
	timeout, err := cfg.TimeoutAsDuration()
	if err != nil {
		return fmt.Sprintf("misconfigured timeout: %v", err)
	}
	return strconv.FormatInt(timeout.Nanoseconds(), 10)
}

// TimeoutAsDuration returns configured timeout as time.Duration.
func (cfg CmdConfig) TimeoutAsDuration() (timeout time.Duration, err error) {
	if nil == cfg.Timeout {
		return CfgDefaultTimeout(), nil
	}

	return ValueDuration(cfg.Timeout)
}

// TimeoutAsDurationOrDefault returns configured timeout as time.Duration, or the default
// timeout in any misconfiguration.
func (cfg CmdConfig) TimeoutAsDurationOrDefault() (timeout time.Duration) {
	timeout, err := cfg.TimeoutAsDuration()
	if nil != err {
		return CfgDefaultTimeout()
	}

	return timeout
}

// StubRequestWith returns a stub request that requests execution of the
// configured cmd with the given arguments.
func (cfg CmdConfig) StubRequestWith(args []string) StubRequest {
	req := StubRequest{
		CmdName: cfg.CmdToStub,
		Args:    args,
		Key:     cfg.StubKey,
	}
	return req
}
