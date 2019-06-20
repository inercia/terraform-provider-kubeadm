package ssh

import (
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
func DoExecList(commands []string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		for _, command := range commands {
			var err error
			if useSudo {
				command = "sudo " + sudoArgs + " " + command
			}

			log.Printf("[DEBUG] [KUBEADM] running '%s'", command)

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
				return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
			}
			waitResult := cmd.Wait()
			if waitResult != nil {
				cmdError, _ := waitResult.(*remote.ExitError)
				if cmdError.ExitStatus != 0 {
					err = fmt.Errorf("Command '%q' exited with non-zero exit status: %d", cmdError.Command, cmdError.ExitStatus)
				}
			}

			outW.Close()
			errW.Close()
			<-outDoneCh
			<-errDoneCh
			return err
		}
		return nil
	})
}

// DoExec is a runner for remote Commands
func DoExec(command string) ApplyFunc {
	return DoExecList([]string{command})
}

// DoExecScript is a runner for a script
func DoExecScript(contents io.Reader, prefix string) ApplyFunc {
	path, err := randomPath(prefix, "sh")
	if err != nil {
		panic(err)
	}

	return ApplyComposed(
		doRealUploadFile(contents, path),
		DoExec(fmt.Sprintf("sh %s", path)),
	)
}

// DoLocalExec executes a local command
func DoLocalExec(command string, args ...string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
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
			return status.Error
		}

		return nil
	})
}

// CheckExec checks if bash command succeedes or not
func CheckExec(cmd string) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		success := "CONDITION_SUCCEEDED"
		failure := "CONDITION_FAILED"
		found := false
		var interceptor OutputFunc = func(s string) {
			// check only the `success` appears, as some other error/log message about
			// the command can contain both...
			if strings.Contains(s, success) && !strings.Contains(s, failure) {
				log.Printf("[DEBUG] Condition succeeded: '%s' found in '%s'", success, s)
				found = true
			}
		}

		command := fmt.Sprintf("%s && echo '%s' || echo '%s'",
			cmd, success, failure)

		log.Printf("[DEBUG] Checking condition: '%s'", cmd)
		err := DoExec(command).Apply(interceptor, comm, useSudo)
		if err != nil {
			return false, err
		}

		return found, nil
	})
}
