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
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/congop/execstub/build/go_generate_stub_exec_assets/diff"
	"github.com/pkg/errors"
)

func main() {
	if len(os.Args) == 0 {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s static|static_com_config|exe",
			os.Args[0])
		fmt.Fprintln(os.Stdout, "Environ:", os.Environ())
		fmt.Fprintln(os.Stdout, "cmd:", os.Args)
		wd, err := os.Getwd()
		fmt.Fprintf(os.Stdout, "wd:%s %v", wd, err)
		os.Exit(math.MaxUint8)
	}

	pjtRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, "fail to get absolute current directory")
		os.Exit(math.MaxUint8)
	}
	if len(os.Args) > 2 {
		pjtRootAbs, err := filepath.Abs(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "fail to get absolute pjtRoot: relPjtRoot=[%s]", pjtRoot)
			os.Exit(math.MaxUint8)
		}
		pjtRoot = pjtRootAbs
	}
	goCmd(pjtRoot, "test", "./build/go_generate_stub_exec_assets/diff")

	genOption := os.Args[1]

	var genFunc func(string) error
	switch genOption {

	case "static":
		genFunc = generateStaticGo
	case "static_com_config":
		genFunc = genrateCmdConfigAssetsGo
	case "exe":
		genFunc = generateExecGo
	default:
		genFunc = func(pjtRoot string) error {
			return errors.Errorf(
				"Unsupported generate option:%s. Usage: %s static|static_com_config|exe",
				genOption, os.Args[0])
		}
	}
	err = genFunc(pjtRoot)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func generateStaticGo(pjtRoot string) error {
	contentSource := "internal/assets/static/bash_exec_stub.sh"
	bashExecStubBase64, hash, err := fileAsBase64(pjtRoot, contentSource)
	if err != nil {
		return errors.WithMessagef(
			err,
			"Failed to read bash_exec_stub.sh file as base64:%s %v",
			contentSource, err)
	}

	data := map[string]interface{}{
		"ExecStubBase64": bashExecStubBase64,
		"ExecStubHash":   hash,
	}
	assetsTmplPath := "internal/assets/static/assets_tmpl.go"
	dstPath := "internal/assets/statics.go"
	err = generateTo(pjtRoot, dstPath, assetsTmplPath, data)
	if err != nil {
		return errors.WithMessagef(
			err, "Fail to generate go code statics.go to: %s, %v", dstPath, err)
	}
	return nil
}

func genrateCmdConfigAssetsGo(pjtRoot string) error {
	contentSource := "internal/assets/static/go_testbinary_tubbing_wrapper.config"
	cmdConfigTmplBase64, _, err := fileAsBase64(pjtRoot, contentSource)
	if err != nil {
		return errors.WithMessagef(
			err,
			"Failed to read go_testbinary_stubbing_wrapper.config file as base64:%s %v",
			contentSource, err)
	}

	data := map[string]interface{}{
		"CmdConfigTmplBase64": cmdConfigTmplBase64,
	}
	assetsTmplPath := "internal/assets/static/assets_tmpl_com_config.go"
	dstPath := "pkg/comproto/cmd_config_assets.go"
	err = generateTo(pjtRoot, dstPath, assetsTmplPath, data)
	if err != nil {
		return errors.WithMessagef(
			err, "Fail to generate go code statics.go to: %s, %v", dstPath, err)
		// os.Exit(1)
	}

	// os.Exit(0)
	return nil
}

func generateExecGo(pjtRoot string) error {
	goCmd(pjtRoot, "build", "./cmd/go_testbinary_tubbing_wrapper")
	goCmd(pjtRoot, "test", "./cmd/go_testbinary_tubbing_wrapper")

	execStubFile := "./go_testbinary_tubbing_wrapper"
	if runtime.GOOS == "windows" {
		execStubFile += ".exe"
	}
	execStubBase64, hash, err := fileAsBase64(pjtRoot, execStubFile)
	if err != nil {
		return errors.WithMessagef(
			err,
			"Failed to read exec stub file as base64:%s %v",
			execStubFile, err)
	}

	assetsTmplPath := "internal/assets/exe/assets_tmpl.go"
	dstPath := "internal/assets/exes.go"
	data := map[string]interface{}{
		"ExecStubBase64": execStubBase64,
		"ExecStubHash":   hash,
	}
	err = generateTo(pjtRoot, dstPath, assetsTmplPath, data)
	if err != nil {
		return errors.WithMessagef(
			err,
			"Fail to generate go code exes.go to: %s, %v", dstPath, err)
	}
	return nil
}

func hashFnv(content []byte) (string, error) {
	hash := fnv.New128a()
	_, err := hash.Write(content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", string(hash.Sum(nil))), nil
}

func fileAsBase64(pjtRoot, path string) (fileContentBase64 string, hash string, err error) {
	pathActual := filepath.Join(pjtRoot, path)
	contentBytes, err := ioutil.ReadFile(pathActual)
	if err != nil {
		dir, _ := os.Getwd()
		return "", "", errors.Wrapf(err,
			"error reading as base64: %s \npjtRoot=%s \nwd=%s --> %s \nenv:%s",
			path, pjtRoot, pathActual, dir, os.Environ())
	}
	hash, err = hashFnv(contentBytes)
	if err != nil {
		return "", "", err
	}
	fileContentBase64 = base64.StdEncoding.EncodeToString(contentBytes)

	return fileContentBase64, hash, nil
}

func generateTo(pjtRoot string, dstPath string, tmplPath string, data interface{}) error {
	var err error
	var pjtRootAbs string
	if pjtRoot != "" {
		pjtRootAbs, err = filepath.Abs(pjtRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to get absolute path for pjtRoot(%s)", pjtRoot)
		}
	}
	tmplPathActual := filepath.Join(pjtRootAbs, tmplPath)
	tmplContent, err := ioutil.ReadFile(tmplPathActual)
	if err != nil {
		return errors.Errorf("failed to read template file %s --> %s %v", tmplPath, tmplPathActual, err)

	}
	generatedHeader := fmt.Sprintf(
		"// Code generated by \"go_generate_stub_exec_assets %s\"; DO NOT EDIT\n"+
			"// generated at %s\n",
		strings.Join(os.Args[1:], " "), time.Now().Format(time.RFC3339Nano))
	// not including \n in string to be replaced because \n character representation
	// dependents on the platform (windows vs. inux / LF vs. CRLF), so that the replacing
	// may not work depending on which platform the template was saved.
	tmplContentNoGoBuildIgnore := strings.Replace(
		string(tmplContent), "// +build ignore", generatedHeader, 1)

	tmpl, err := template.New("xxx").Parse(tmplContentNoGoBuildIgnore)
	if err != nil {
		return errors.Errorf(
			"failed to load template %s \n\terr:%v \n\ttmplContentNoGoBuildIgnore:%s",
			tmplPath, err, tmplContentNoGoBuildIgnore)

	}
	dstTmpFileName := ""
	{
		dstTmpFile, err := ioutil.TempFile("", "go_generate_stub_exec_assets_gen_*.go")
		if err != nil {
			return err
		}
		if err := tmpl.Execute(dstTmpFile, data); err != nil {
			dstTmpFile.Close()
			return err
		}
		// close to flush content so that we can go fmt
		// sync() seems not to work
		dstTmpFile.Close()
		dstTmpFileName = dstTmpFile.Name()
		err = gofmt(dstTmpFileName)
		if err != nil {
			return err
		}
	}

	defer func() {
		if err != nil {
			os.Remove(dstTmpFileName)
		}
	}()

	dstPathActual := filepath.Join(pjtRootAbs, dstPath)
	eq, err := diff.EqualIgnoreLineCommentPath(dstTmpFileName, dstPathActual)
	if err != nil {
		return errors.Wrapf(err,
			"fail to diff ignoring line comment of new and old content "+
				"\n\ttmp file new content=%s \n\told content=%s",
			dstTmpFileName, dstPathActual)
	}

	if eq {
		fmt.Printf(
			"Skip update of %s because current content does not difer from"+
				" new content (%s) when ignoring line comments\n",
			dstPathActual, dstTmpFileName)
		return nil
	}

	err = os.Rename(dstTmpFileName, dstPathActual)
	if err != nil {
		err = errors.Wrapf(
			err,
			"fail to replace old with new content \n\tnew content path=%s \n\tdestination=%s",
			dstTmpFileName, dstPathActual)
		return err
	}

	fmt.Printf("Go file generated at:%s\n", dstPath)
	return nil
}

func gofmt(goFile string) error {
	if filepath.Ext(goFile) != ".go" {
		return errors.Errorf(
			"will only format a single go file: extension: %s filepath:%s",
			filepath.Ext(goFile), goFile)
	}
	gofmtCmd := exec.Command("go", "fmt", goFile)
	stdoutErr, err := gofmtCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err,
			"failed formating go file: %s \nerr=%v \nstdoutErr=%s",
			goFile, err, stdoutErr)

	}
	return nil
}

func goCmd(pjtRoot, cmd, packageRelPath string) {
	exePackageAll := packageRelPath + "/..."
	goBuildExe := exec.Command("go", cmd, "-v", exePackageAll)
	goBuildExe.Dir = pjtRoot
	stdoutErr, err := goBuildExe.CombinedOutput()
	if err != nil {
		fmt.Printf(
			"failed to go %s package:%s \nstdOutErr=%s \nerr=%v",
			cmd, exePackageAll, string(stdoutErr), err)
		os.Exit(1)
	}
}
