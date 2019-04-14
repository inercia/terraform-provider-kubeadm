package kubeadm

import (
	"fmt"
	"io"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

// remoteCommands is a runner for remote commands
type remoteCommands struct {
	Output terraform.UIOutput
	Comm   communicator.Communicator
}

// newRemoteCommands creates a ne runner for remote commands
func newRemoteCommands(o terraform.UIOutput, comm communicator.Communicator) remoteCommands {
	return remoteCommands{Output: o, Comm: comm}
}

// Run runs a list of commands
func (rc remoteCommands) Run(commands []string, useSudo bool) error {
	for _, command := range commands {
		var err error
		if useSudo {
			command = "sudo " + command
		}

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

		go copyOutput(rc.Output, outR, outDoneCh)
		go copyOutput(rc.Output, errR, errDoneCh)

		cmd := &remote.Cmd{
			Command: command,
			Stdout:  outW,
			Stderr:  errW,
		}

		rc.Output.Output(fmt.Sprintf("running command: %s", command))
		if err := rc.Comm.Start(cmd); err != nil {
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
}

// /////////////////////////////////////////////////////////////////////////////////

// A script that will be run remotely
type remoteScript struct {
	remoteFile
	path string
}

func newRemoteScript(o terraform.UIOutput, comm communicator.Communicator) remoteScript {
	return remoteScript{
		remoteFile: remoteFile{o, comm},
		path:       "",
	}
}

func (rs remoteScript) Run(args string, useSudo bool) error {
	cmds := []string{fmt.Sprintf("%s %s", rs.path, args)}
	return newRemoteCommands(rs.Output, rs.Comm).Run(cmds, false)
}

// Perform a cleanup by removing the remote file
func (rs remoteScript) Cleanup() error {
	cmds := []string{fmt.Sprintf("rm -f %s", rs.path)}
	return newRemoteCommands(rs.Output, rs.Comm).Run(cmds, false)
}
