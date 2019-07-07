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
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	// arguments for "sudo"
	sudoArgs = "--non-interactive"
)

// DoExecList is a runner for a list of remote commands
func DoExecList(commands []string) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		var ae ActionError
		for _, command := range commands {
			if len(command) == 0 {
				continue
			}

			if useSudo {
				command = "sudo " + sudoArgs + " " + command
			}

			log.Printf("[DEBUG] [KUBEADM] running '%s'", command)

			// TODO: reuse the same pipe's and stuff between commands
			outR, outW := io.Pipe()
			errR, errW := io.Pipe()
			outDoneCh := make(chan struct{})
			errDoneCh := make(chan struct{})

			copyOutput := func(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
				defer close(doneCh)
				lr := linereader.New(r)
				for line := range lr.Ch {
					o.Output(line)
				}
			}

			go copyOutput(o, outR, outDoneCh)
			go copyOutput(o, errR, errDoneCh)

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
					ae = ActionError(fmt.Sprintf("Command '%q' exited with non-zero exit status: %d", cmdError.Command, cmdError.ExitStatus))
				}
			}

			outW.Close()
			errW.Close()
			<-outDoneCh
			<-errDoneCh
		}
		return ae
	})
}

// DoExec is a runner for remote Commands
func DoExec(command string) Action {
	return DoExecList([]string{command})
}

// DoExecScript is a runner for a script (with some random path in /tmp)
func DoExecScript(contents io.Reader, prefix string) Action {
	path, err := GetTempFilename()
	if err != nil {
		return ActionError(fmt.Sprintf("Could not create temporary file: %s", err))
	}

	return DoWithCleanup(
		ActionList{
			doRealUploadFile(contents, path),
			DoExec(fmt.Sprintf("sh %s", path)),
		},
		DoDeleteFile(path),
	)
}

// DoLocalExec executes a local command
func DoLocalExec(command string, args ...string) Action {
	return ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) Action {
		fullCmd := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
		o.Output(fmt.Sprintf("Running local command %q...", fullCmd))

		// Disable output buffering, enable streaming
		cmdOptions := cmd.Options{
			Buffered:  false,
			Streaming: true,
		}

		envCmd := cmd.NewCmdOptions(cmdOptions, command, args...)

		go func() {
			for {
				select {
				case line := <-envCmd.Stdout:
					o.Output(line)
				case line := <-envCmd.Stderr:
					o.Output("ERROR: " + line)
				}
			}
		}()

		// Run and wait for Cmd to return
		status := <-envCmd.Start()

		// Cmd has finished but wait for goroutine to print all lines
		for len(envCmd.Stdout) > 0 || len(envCmd.Stderr) > 0 {
			time.Sleep(10 * time.Millisecond)
		}

		if status.Exit != 0 {
			o.Output(fmt.Sprintf("Error waiting for %q: %s [%d]",
				command, status.Error.Error(), status.Exit))
			return ActionError(status.Error.Error())
		}

		return nil
	})
}

// CheckExec checks if bash command succeedes or not
func CheckExec(cmd string) CheckerFunc {
	const success = "CONDITION_SUCCEEDED"
	const failure = "CONDITION_FAILED"
	command := fmt.Sprintf("%s && echo '%s' || echo '%s'", cmd, success, failure)

	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		log.Printf("[DEBUG] [KUBEADM] Checking condition: '%s'", cmd)
		var buf bytes.Buffer
		if res := DoSendingOutputToWriter(DoExec(command), &buf).Apply(o, comm, useSudo); IsError(res) {
			log.Printf("[DEBUG] [KUBEADM] ERROR: check failed: %s", res)
			return false, res
		}

		// check _only_ the `success` appears, as some other error/log message about
		// the command can contain both...
		s := buf.String()
		if strings.Contains(s, success) && !strings.Contains(s, failure) {
			log.Printf("[DEBUG] [KUBEADM] Condition %q succeeded: %q found", cmd, success)
			return true, nil
		}
		log.Printf("[DEBUG] [KUBEADM] Condition %q failed", cmd)
		return false, nil
	})
}
