// Copyright © 2019 Alvaro Saurin
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

package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	// arguments for "sudo"
	sudoArgs = "--non-interactive -E"

	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

func copyOutput(output terraform.UIOutput, input io.Reader, done chan<- struct{}) {
	defer close(done)
	lr := linereader.New(input)
	for line := range lr.Ch {
		output.Output(line)
	}
}

// DoExec is a runner for remote Commands
func DoExec(command string) Action {
	return ActionFunc(func(ctx context.Context) (res Action) {
		if len(command) == 0 {
			return nil
		}

		execOutput := GetExecOutputFromContext(ctx)
		comm := GetCommFromContext(ctx)

		if GetUseSudoFromContext(ctx) {
			command = "sudo " + sudoArgs + " " + command
		}

		Debug("running %q", command)

		outR, outW := io.Pipe()
		errR, errW := io.Pipe()
		outDoneCh := make(chan struct{})
		errDoneCh := make(chan struct{})

		go copyOutput(execOutput, outR, outDoneCh)
		go copyOutput(execOutput, errR, errDoneCh)

		cmd := &remote.Cmd{
			Command: command,
			Stdout:  outW,
			Stderr:  errW,
		}

		if err := comm.Start(cmd); err != nil {
			return ActionError(fmt.Sprintf("Error executing command %q: %v", cmd.Command, err))
		}
		waitResult := cmd.Wait()
		if waitResult != nil {
			cmdError, _ := waitResult.(*remote.ExitError)
			if cmdError.ExitStatus != 0 {
				msg := fmt.Sprintf("Command %q exited with non-zero exit status: %d", cmdError.Command, cmdError.ExitStatus)
				Debug(msg)
				res = ActionError(msg)
			}
			// otherwise, it is a communicator error
		}

		_ = outW.Close()
		_ = errW.Close()

		select {
		// wait until the copyOutput function is done (for stdout and the stderr)
		case <-outDoneCh:
			<-errDoneCh
		// .. or until the context is done
		case <-ctx.Done():
		}

		return
	})
}

// DoExecScript is a runner for a script (with some random path in /tmp)
func DoExecScript(contents []byte) Action {
	path, err := GetTempFilename()
	if err != nil {
		return ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
	}
	return DoWithCleanup(
		ActionList{
			doRealUploadFile(contents, path),
			DoExec(fmt.Sprintf("sh %s", path)),
		},
		ActionList{
			DoTry(DoDeleteFile(path)),
		})
}

// DoLocalExec executes a local command
func DoLocalExec(command string, args ...string) Action {
	return ActionFunc(func(ctx context.Context) Action {
		userOutput := GetUserOutputFromContext(ctx)
		execOutput := GetExecOutputFromContext(ctx)

		fullCmd := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
		userOutput.Output(fmt.Sprintf("Running local command %q...", fullCmd))

		// Setup the reader that will read the output from the command.
		// We use an os.Pipe so that the *os.File can be passed directly to the
		// process, and not rely on goroutines copying the data which may block.
		// See golang.org/issue/18874
		pr, pw, err := os.Pipe()
		if err != nil {
			return ActionError(fmt.Sprintf("failed to initialize pipe for output: %s", err))
		}

		//var cmdEnv []string
		//cmdEnv = os.Environ()
		//cmdEnv = append(cmdEnv, env...)

		// Setup the command
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Stderr = pw
		cmd.Stdout = pw

		// Dir specifies the working directory of the command.
		// If Dir is the empty string (this is default), runs the command
		// in the calling process's current directory.
		//cmd.Dir = workingdir

		// Env specifies the environment of the command.
		// By default will use the calling process's environment
		//cmd.Env = cmdEnv

		output, _ := circbuf.NewBuffer(maxBufSize)

		// Write everything we read from the pipe to the output buffer too
		tee := io.TeeReader(pr, output)

		// copy the teed output to the UI output
		copyDoneCh := make(chan struct{})
		go copyOutput(execOutput, tee, copyDoneCh)

		// Start the command
		err = cmd.Start()
		if err == nil {
			err = cmd.Wait()
		}

		// Close the write-end of the pipe so that the goroutine mirroring output
		// ends properly.
		_ = pw.Close()

		// Cancelling the command may block the pipe reader if the file descriptor
		// was passed to a child process which hasn't closed it. In this case the
		// copyOutput goroutine will just hang out until exit.
		select {
		case <-copyDoneCh:
		case <-ctx.Done():
		}

		if err != nil {
			msg := fmt.Sprintf("Error running command '%s': %v", command, err)
			return ActionError(msg)
		}
		return nil
	})
}

// CheckExec checks if bash command succeedes or not
func CheckExec(cmd string) CheckerFunc {
	const success = "CONDITION_SUCCEEDED"
	const failure = "CONDITION_FAILED"
	command := fmt.Sprintf("%s && echo '%s' || echo '%s'", cmd, success, failure)

	return CheckerFunc(func(ctx context.Context) (bool, error) {
		Debug("Checking condition: '%s'", cmd)
		var buf bytes.Buffer
		if res := DoSendingExecOutputToWriter(DoExec(command), &buf).Apply(ctx); IsError(res) {
			Debug("ERROR: when performing check %q: %s", cmd, res)
			return false, res
		}

		// check _only_ the `success` appears, as some other error/log message about
		// the command can contain both...
		s := buf.String()
		Debug("check: output: %q", s)
		if strings.Contains(s, success) && !strings.Contains(s, failure) {
			Debug("check: %q succeeded (%q found in output)", cmd, success)
			return true, nil
		}
		Debug("check: %q failed", cmd)
		return false, nil
	})
}

// CheckBinaryExists checks that a binary exists in the path
func CheckBinaryExists(cmd string) CheckerFunc {
	// note: start 'command' in a subshell, as it doesn't mix well with 'sudo'
	command := fmt.Sprintf("sh -c \"command -v '%s'\"", cmd)

	return CheckerFunc(func(ctx context.Context) (bool, error) {
		Debug("Checking binary exists with: '%s'", cmd)
		var buf bytes.Buffer
		if res := DoSendingExecOutputToWriter(DoExec(command), &buf).Apply(ctx); IsError(res) {
			Debug("ERROR: when performing check: %s", res)
			return false, res
		}

		// if "command -v" doesn't print anything, it was not found
		s := strings.TrimSpace(buf.String())
		if s == "" {
			Debug("%q NOT found: empty output: output == %q", cmd, s)
			return false, nil
		}

		// sometimes it just returns the file name provided
		if s == cmd {
			Debug("%q found: output == %q", cmd, s)
			return true, nil
		}

		// if it prints the full path: check it is really there
		if path.IsAbs(s) {
			Debug("checking file %q exists at %q", cmd, s)
			return CheckOnce(
				fmt.Sprintf("path-command-%s", s),
				CheckFileExists(s)).Check(ctx)
		}

		// otherwise, just fail
		Debug("%q NOT found: output == %q", cmd, s)
		return false, nil
	})
}
