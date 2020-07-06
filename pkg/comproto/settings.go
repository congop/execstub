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

// DiscoveredBy specifies how executable is discovered.
type DiscoveredBy string

const (
	// DiscoveredByPath specifies that the command is discovered using environment path
	DiscoveredByPath = "PATH"

	// DiscoveredByHomeBinDir specifies that the command is discovered using
	// an home directory setting in environment.
	DiscoveredByHomeBinDir = "HomeBinDir"
)

// ExecType specifies the type of exec used for stubbing
type ExecType string

const (
	// ExecTypeExe requires the use of an Executable for the stubbing
	ExecTypeExe = "EXE"

	// ExecTypeBash requires the use of a bash script for the stubbing
	ExecTypeBash = "BASH"
)

// StubbingMode specifies the stubing mode (static or dynamic).
type StubbingMode string

const (
	// StubbingModeStatic requires static stubbing.
	// The stubfunc will be evaludated immediately (before the command execution)
	// with zero request.
	// The outcome is static and cannot  depend on the command arguments
	StubbingModeStatic = "STATIC"

	// StubbingModeDyna reuqires dynamic stubbing.
	// The stub-func will be evaluated online (reacting to the command execution)
	// The outome ca there for depends on command arguments
	StubbingModeDyna = "DYNAMIC"
)

// DiscoveredByHomeDirBinData holds data used to the home environment key and
// the binary sub-directory structure which contains the executable
// e.g. For Java: EnvHomeKey=JAVA_HOME and BinDirs=[]string{"bin"}
// 			so that the java binary is located in ${JAVA_HOME}/bin/
type DiscoveredByHomeDirBinData struct {
	EnvHomeKey string
	// BinDirs bin sub directory paths.
	// using to avoid guessing path separator
	BinDirs []string
}

// Settings holds setting data for an exec stubbing
type Settings struct {
	DiscoveredBy               DiscoveredBy
	DiscoveredByHomeDirBinData DiscoveredByHomeDirBinData

	ExecType ExecType

	Mode StubbingMode

	TestHelperProcessMethodName string

	Timeout OptionalDuration
}

// SettingsDynaStubCmdDiscoveredByPath constructs a new Settings for dynamically stubbing
// a cmd which is discovered by Path.
func SettingsDynaStubCmdDiscoveredByPath() *Settings {
	s := Settings{}
	s.DiscoveredByPath()
	s.ExecTypeExe()
	s.ModeDanymic()
	s.WithoutTestProcessHelper()
	return &s
}

// IsCmdDiscoveredByHomeDir true is cmd are set to be discovered by home-dir.
func (s Settings) IsCmdDiscoveredByHomeDir() bool {
	return DiscoveredByHomeBinDir == s.DiscoveredBy
}

// DiscoveredByPath selects discovery by path
func (s *Settings) DiscoveredByPath() {
	s.DiscoveredBy = DiscoveredByPath
	s.DiscoveredByHomeDirBinData.BinDirs = []string{}
	s.DiscoveredByHomeDirBinData.EnvHomeKey = ""
}

// DiscoveredByHomeDirBin selects discovery by home dir
func (s *Settings) DiscoveredByHomeDirBin(envHomeKey string, binDirs ...string) {
	s.DiscoveredBy = DiscoveredByHomeBinDir
	s.DiscoveredByHomeDirBinData.BinDirs = binDirs
	s.DiscoveredByHomeDirBinData.EnvHomeKey = envHomeKey
}

// ExecTypeBash sets exec type to BASH
func (s *Settings) ExecTypeBash() {
	s.ExecType = ExecTypeBash
}

// ExecTypeExe sets exec type to EXE
func (s *Settings) ExecTypeExe() {
	s.ExecType = ExecTypeExe
}

// ModeStatic set stubbing mode to static
func (s *Settings) ModeStatic() {
	s.Mode = StubbingModeStatic
}

// ModeDanymic set stubbing mode to dynamic
func (s *Settings) ModeDanymic() {
	s.Mode = StubbingModeDyna
}

// WithTestProcessHelper requires the use a test process helper.
func (s *Settings) WithTestProcessHelper(methodName string) {
	s.TestHelperProcessMethodName = methodName
}

// WithoutTestProcessHelper requires stubbing without the use of a test process helper
func (s *Settings) WithoutTestProcessHelper() {
	s.TestHelperProcessMethodName = ""
}

// InModStatic returns true if in mode static false otherwise.
func (s Settings) InModStatic() bool {
	return StubbingModeStatic == s.Mode
}

// InModDyna returns true if in mode dynamic false otherwise.
func (s Settings) InModDyna() bool {
	return StubbingModeDyna == s.Mode
}

// IsUsingTestProcessHelper returns true if TestProcessHelper is provided
// and expected to be used; false otherwise.
func (s Settings) IsUsingTestProcessHelper() bool {
	return s.TestHelperProcessMethodName != ""
}

// IsUsingExecTypeBash returns true bash is specified as exec type; false otherwise.
func (s Settings) IsUsingExecTypeBash() bool {
	return ExecTypeBash == s.ExecType
}
