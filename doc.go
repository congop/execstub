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

/*Package execstub provides stubbing for a usage of a command line api which is
based on an executable discovered using the environment <PATH>
or some sort of <HOME>+<Bin> configuration.

  - What is a usage of a command line api?
    Its when you spawn a sub-process using a programming language api.
    Think go exec.Cmd.Run(..), java Runtime.exec(..), python subprocess.run(..).
    The sub-process execution wil yield an outcome consisting of:
 		  - side effects (e.g. a file created when executing touch).
      - bytes or string in std-out and/or std-err.
      - an exit code.
    The executed binary is tipically specified using a path like argument.
    It can be:
      - a absolute path
      - a relative path
      - the name of the binary

  - Discovering the executable
    How the binary is discovered depends on the art the specified path like argument.
    The goal of the discovery process is to transform the path like specification
    into an absolute path, so that if the specification is done with:
    - absolute path
      nothing is left to be done
    - relative path
      it will be appended to the current directory to form an absolute path
    - name of the binary
      a file with that name will be search in the directories specified
      by the PATH environment variable. The first found will be used.
      On some platforms known and specified file exetensions (e.g. .exe) will be appended
      to the name during the search.
    Even when using absolute paths, you may decide to put a customization strategy
    using some process environment variable. We call this the home-bin-dir strategy.
    E.g. Java executed base on JAVA_HOME or JRE_HOME with the java executable located
    at ${JAVA_HOME}/bin/java

  - Stubbing mechanism
    The aim os stubbing is to replace at execution time the actual executable with
    a fake one which will yield some kind of pre-defined known outcome. This can be done
    if there is step in the binary discovery which can be tweaked so that the stud-executable
    is discovered instead of the actual one.
    It is the case if binary is specified by name and discovered using PATH or
    when absolute path is used but with a customization layer in place based
    on defined environement home variable.
    The mechanism will just have to mutate the PATH or set the home variable accordingly.
    Stubbing become complicated if the specification is based on absolute (without customization
    layer) or relative path because it will require e.g.:
      - a file overley to hide the original executable
      - a change of the current directory
      - an in the flight replacement of the original executable and a rollback.
    These are either not desirable, or too heavy especially in a unit-test-ish context.
    The execstub module will therefore only provide stubbing for PATH and Home-Bin-Dir based discovery.

  - Design elements
    - Modelling the command line usage
      We have basically 3 aspects to model:
        - Starting the process
          It is modelled using StubRequest which hold the the command name and the
          execution argument so that the fake execution process can be function of them.

        - Execution
          The outcome can be a static sum of known in advance stderr, stdout and exit code.
          This is referred to as static mode.
          It may also be desirale to have some side effects, e.g.:
            - related to the execution itself like creating a file
            - or related to the unit-test as counting the number of calls
          In other constelations the outocme may requires some computation to determine
          stderr/stdout/exicode as function of the request.
          We refer to these cases as dynamic mode.
          The execution is therefore modelled as function namely StubFunc.
          Thus, if required the StubFunc corresponding to the current stubbing will be looked-up and run.

        - Outcome
          The stderr/stdout/exit-code part of the outcome is modelled as ExecOutcome.
          The execution the the stubfunc may result however in an error.
          Such error is not an actual sub-process execution erronuous outcome.
          It therefore must be modelled separately as opposed to be added to stderr
          and setting a non-zero exit code.
          Such and error can be communicated by using the ExecOutcome field InternalErrTxt.

    - Stub-Executable
      It is the fake executable used to effectuate the overall outcome.
      It may realise the outcome by itself. It may also cooperate with a test helper
      to achieve the outcome. The following command line is used:
      /tmp/go-build720053430/b001/xyz.test -test.run=TestHelperProcess -- arg1 arg2 arg3
      It is not possible to use the test helper directly because we will need to inject
      the args test.run and -- at the actual execution call site.
      Two kinds of stub-executable are provided:
        - bash based, which uses a bash script
        - exec base, which used an go based executable
      Obviously the go based one is bound to support more platform than the bash based one.
      Both executable are genric and need context information about the stubbing for which
      they will be used. This context data is modeled as CmdConfig and saved as file
      alongside the executable.

    - Inter process communication (IPC) to support dynamic outcomes
      The invokation  of a StubFunc to produce a dynamic outcome is done in the unit test process.
      There is no means to access the stub function directly from the fake process execution.
      IPC is used here to allow the fake process to issue a StubRequest  and receive an ExecOutcome.
      The Serialization/Deserialization mechanism needed for IPC must be understood
      by both parties (mind bash) and satisfy a domain specific requirement of
      been able to cope with multiline sterr and stdout. We choose a combination of :
        - base64 to encode the discrete data (e.g. stderr, stdout)
        - and comma separated value (easily encoded and decoded in bash) for envelope encoding
      Named pipes are used for the actual data transport, because they are file based and easily
      handle in different setups (e.g. in bash).

    - Stubbing Setup
      An ExecStubber is provided to manage the stubbing setup.
      Its key feature is to setup an invokation of a StubFunc to replace the execution
      of a process, See ExecStubber.WhenExecDoStubFunc(...).
      It allows a baterie of settings beyond the basic requirement of specifying
      the executable to be stubbed and a StubFunc replacement.
      This is modelled using Settings, which provides the following configuration options:
        - Selecting and customizing the discovery mode (PATH vs. Home-Bin-Dir)
        - Selecting the stub executable (Bash vs. Exec(go based))
        - Seclecting the stubbing mode (static vs. dynamic)
        - Specifying the test process helper method name
        - specifying a timeout  for a stub sub-process execution
      A stubbing setup is identified by a key.
      The key can be use to:
        - unwind the setup(Unregister)
        - access the stubbing data basis of static requests
          (FindAllPersistedStubRequests, DeleteAllPersistedStubRequests ).

    - Conccurency and parallelism
      The mutation of the process environment is the key enabling mechanism of Execstub.
      The process environment must therefore be guarded against issues of concurrency
      and parallelism. This also mean that is not possible to have a parallel stubbing
      setup for the same executable within a test process. The outcome will otherwise be
      non-deterministic because the setting will likely override each other.
      ExecStubber provides a locking mechanism to realize the serialization of the mutation of environment.
      For this to work correctly however there must only be one ExecStubber per unit test process.
      Note that the code under test can still execute its sub-processes concurently or in parallel.
      The correctnes of the outcome here dependents on the implementation of the StubFunc
      function being used. Static outcomes without side effect are of course always deterministic.

*/
package execstub

//TODO use https://godoc.org/bitbucket.org/avd/go-ipc/fifo as fifo implementation
