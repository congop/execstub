# Execstub

Package execstub provides stubbing for a usage of a command line API which is
based on an executable discovered using the environment <PATH>
or some sort of <HOME>+<Bin> configuration.
An example of command line api usage is:
```go
cmd := exec.Command("genisoimage", "-output", cloudInitPath, "-V", "cidata", "-joliet", "-rock", nocloudDir)
	stdInOutBytes, err := cmd.CombinedOutput()
```
Please check out Execstub package doc for more details about the concepts being used.


`go get github.com/congop/execstub`

[Read the package documentation for more information](https://godoc.org/github.com/congop/execstub).

## Usage
```go
import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	rt "github.com/congop/execstub/internal/runtime"
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
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, rt.EnsureHasExecExt("SuperExe"), comproto.Settings{})
	ifNotEqPanic(nil, err, "fail to setup stub")

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	// accessing and checking stubrequest dynymic mode
	gotRequests := *reqStore
	wanRequets := []comproto.StubRequest{
		{
			CmdName: rt.EnsureHasExecExt("SuperExe"), Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, gotRequests, "unexpected stub requests")

	// accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

// used as test process help Settings{TestHelperProcessMethodName: "TestHelperProcExample_dynamic"}
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
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, rt.EnsureHasExecExt("docker"), setting)
	ifNotEqPanic(nil, err, "fail to setip stub")

	// args := []string{"image", "ls", "--format", "\"{{.Repository}}:{{.Tag}}\""}
	args := []string{"image", "ls", "--format", "table '{{.Repository}}:{{.Tag}}'"}
	cmd := exec.Command("docker", args...)
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "exit code set to 0 ==> execution should succeed")

	// accessing and checking stubrequest dynymic mode
	gotRequests := *reqStore
	wanRequets := []comproto.StubRequest{
		{
			CmdName: rt.EnsureHasExecExt("docker"), Args: args, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, gotRequests, "unexpected stub requests")

	// accessing and checking outcome
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
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, rt.EnsureHasExecExt("SuperExe"), settings)
	ifNotEqPanic(nil, err, "fail to setup stub")
	ifNotEqPanic(
		[]comproto.StubRequest{{Key: key, CmdName: rt.EnsureHasExecExt("SuperExe"), Args: nil}},
		*reqStore,
		"Static mod evaluate StubFunc at setup with a stubrequest havin nil args")
	*reqStore = (*reqStore)[:0]

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer
	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	// accessing and checking stubrequest static mode
	ifNotEqPanic(0, len(*reqStore), "Unexpected StubFunc call in static mode")
	gotRequests, err := stubber.FindAllPersistedStubRequests(key)
	ifNotEqPanic(nil, err, "fail to find all persisted stub request")
	wanRequets := []comproto.StubRequest{
		{
			CmdName: rt.EnsureHasExecExt("SuperExe"), Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, *gotRequests, "unexpected stub requests")

	// accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr")

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}

// TestHelperProcExample_static is used as test process help .
// Configured wihh: Settings{TestHelperProcessMethodName: "TestHelperProcExample_static"}
func TestHelperProcExample_static(t *testing.T) {
	extraJobOnStubRequest := func(req comproto.StubRequest) error {
		// some extrat side effect
		// we are adding to stdout
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
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, rt.EnsureHasExecExt("SuperExe"), settings)
	ifNotEqPanic(nil, err, "fail to setip stub")
	ifNotEqPanic(
		[]comproto.StubRequest{{Key: key, CmdName: rt.EnsureHasExecExt("SuperExe"), Args: nil}},
		*reqStore,
		"Static mod evaluate StubFunc at setup with stubrequest having nil args")
	*reqStore = (*reqStore)[:0]

	cmd := exec.Command("SuperExe", "arg1", "argb")
	var bufStderr, bufStdout bytes.Buffer
	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	// accessing and checking stubrequest static mode
	ifNotEqPanic(0, len(*reqStore), "Unexpected StubFunc call in static mode")
	gotRequests, err := stubber.FindAllPersistedStubRequests(key)
	ifNotEqPanic(nil, err, "fail to find all persisted stub request")
	wanRequets := []comproto.StubRequest{
		{
			CmdName: rt.EnsureHasExecExt("SuperExe"), Args: []string{"arg1", "argb"}, Key: key,
		},
	}
	ifNotEqPanic(wanRequets, *gotRequests, "unexpected stub requests")

	// accessing and checking outcome
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
		ExecType: comproto.ExecTypeExe,
	}
	key, err := stubber.WhenExecDoStubFunc(recStubFunc, rt.EnsureHasExecExt("java"), settings)
	ifNotEqPanic(nil, err, "fail to setip stub")

	javaCmd := os.ExpandEnv("${JAVA_HOME}/bin/java")
	cmd := exec.Command(javaCmd, "-version")
	var bufStderr, bufStdout bytes.Buffer

	cmd.Stderr = &bufStderr
	cmd.Stdout = &bufStdout

	err = cmd.Run()
	ifNotEqPanic(nil, err, "should have hat successful execution")

	// accessing and checking stubrequest dynymic mode

	wantRequests := []comproto.StubRequest{
		{
			CmdName: rt.EnsureHasExecExt("java"), Args: []string{"-version"}, Key: key,
		},
	}
	ifNotEqPanic(wantRequests, *reqStore, "unexpected stub requests")

	// accessing and checking outcome
	gotStderr := bufStderr.String()
	ifNotEqPanic(staticOutcome.Stderr, gotStderr, "unexpected stderr //stdout:"+bufStdout.String())

	gotStdout := bufStdout.String()
	ifNotEqPanic(staticOutcome.Stdout, gotStdout, "unexpected stdout")

	gotExitCode := cmd.ProcessState.ExitCode()
	ifNotEqPanic(int(staticOutcome.ExitCode), gotExitCode, "unexpected exec exit code")

	// Output:
}
...
```
See [stubbing_example_test.go](./stubbing_example_test.go)

## Why is Bash exec type still an option
I started the journey by using bash for a quick implementation.
At some point I moved the stub code in a standalone more serious package.
But I kept bash as a challenge (can I do that in bash!?!)
I suggest you prefer the Exec base stub, which is the default if you do not set:
```go
comproto.Settings.ExecType
```

## Alternatives:
I believe in having options/choices. So use this package whenever you pleases.
You should however be aware of the following alternatives to or for CLI API usage:
  - Use package variable representing the command which you can change in unit-test.
  - Build an abstraction which models the execution as a strategy.
    when unit-test just plug the fake behavior
  - Build an abstraction around the client lib, which uses the command-line-api.
  - Use an available binding for your language
  - Use an available "remote" API (think HTTP REST, WSDL, gRPC)

## Limitation
- Supported platform<br/>
  This project is been developed on Ubuntu 18.04 and tested on both ubuntu and alpine linux.
  It should work fine on any linux(-ish) system having an up-to-date bash, and coretools installed.
  The windows support is in experimental state. Note the following about it:
  - Opting to use the bash based stub executable.
   (bash is not a native windows tool and the <#!-make-it-executable> is missing)
  - IPC not implemented with named pipes
    Named pipes implementation in windows requires a synchronous client-server interaction flow.
		It does not support the simple open / write mechamism as in Linux.
		Therefore an implementation based on normal file open/write/read has been implemented instead.
- Timeout implementation<br/>
  It is possible to set a timeout for the execution of the stubbing sub-process.
  Please note that the enforcement of timeout is very inaccurate.
- Parallelism and Concurrency<br/>
  The mutation of the process environment is the key enabling mechanism of Execstub.
  To avoid concurrency issues you must stick to one ExecStubber per test process.
  There must at any time be one stubbing set-up for stubbing a command which is uniquely
  identified by a name.
  This does not impose a concurrency or parallelism limitation on how the code under test
  executes sub-processes. In this case the determinism of the outcome is determined by the
  implementation of the StubFunc, which is used to effectuate the specified outcome.

## Contributing

You are welcome to add more greatness to this project.
Here some things you could do:
- use it
- write about it
- give your feedback
- reports an issue
- suggest a missing feature
- send pull requests (please discuss your change first by raising an issue)
- etc.

## License

[Apache 2.0 license](./LICENSE)
