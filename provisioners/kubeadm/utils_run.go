package kubeadm

// note: we could use go-bindata too, but that would mean we would have to include an extra dependency
//       just for this... I think it is not worth.
//go:generate ../../utils/generate.sh --out-var setupScriptCode --out-package kubeadm --out-file generated_setup.go ./static/kubeadm-setup.sh
//go:generate ../../utils/generate.sh --out-var kubeletMasterCode --out-package kubeadm --out-file generated_kubelet_master.go ./static/kubelet_master
//go:generate ../../utils/generate.sh --out-var kubeletNodeCode --out-package kubeadm --out-file generated_kubelet_node.go ./static/kubelet_node

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	defaultRemoteTmp = "/tmp"

	// the path where a kubeadm runner will be installed/linked
	defaultKubeadmRunner = "/tmp/kubeadm.sh"
)

func randBytes(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomPath(prefix, extension string) (string, error) {
	r, err := randBytes(3)
	if err != nil {
		return "", err
	}
	if len(prefix) == 0 || len(extension) == 0 {
		return "", fmt.Errorf("can not use empty prefix or extension")
	}
	return fmt.Sprintf("%s/%s-%s.%s", defaultRemoteTmp, prefix, r, extension), nil
}

type remoteFile struct {
	Output terraform.UIOutput
	Comm   communicator.Communicator
	Path   string
}

func newRemoteFile(o terraform.UIOutput, comm communicator.Communicator) remoteFile {
	return remoteFile{Output: o, Comm: comm}
}

func (rs *remoteFile) Upload(contents io.Reader, prefix, extension string) error {
	p, err := randomPath(prefix, extension)
	if err != nil {
		return err
	}
	return rs.UploadTo(contents, p)
}

func (rs *remoteFile) UploadTo(contents io.Reader, p string) error {
	rs.Path = p
	return rs.Comm.Upload(rs.Path, contents)
}

func (rs *remoteFile) UploadScript(contents io.Reader, prefix string) error {
	p, err := randomPath(prefix, "sh")
	if err != nil {
		return err
	}
	rs.Path = p
	return rs.Comm.UploadScript(rs.Path, contents)
}

// Perform a cleanup by removing the remote file
func (rs remoteFile) Cleanup() error {
	return runCommand(rs.Output, rs.Comm, false, fmt.Sprintf("rm -f %s", rs.Path))
}

// A script that will be run remotely
type remoteScript struct {
	remoteFile
}

func newRemoteScript(o terraform.UIOutput, comm communicator.Communicator) remoteScript {
	return remoteScript{remoteFile{Output: o, Comm: comm}}
}

func (rs remoteScript) Run(args string, useSudo bool) error {
	return runCommand(rs.Output, rs.Comm, useSudo, fmt.Sprintf("%s %s", rs.Path, args))
}

func runCommands(o terraform.UIOutput, comm communicator.Communicator, useSudo bool, commands []string) error {
	for _, command := range commands {
		if err := runCommand(o, comm, useSudo, command); err != nil {
			return err
		}
	}
	return nil
}

// Run a command in the remote resource
func runCommand(o terraform.UIOutput, comm communicator.Communicator, useSudo bool, command string) error {
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

	go copyOutput(o, outR, outDoneCh)
	go copyOutput(o, errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	o.Output(fmt.Sprintf("running command: %s", command))
	if err := comm.Start(cmd); err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}
	cmd.Wait()
	if cmd.ExitStatus != 0 {
		err = fmt.Errorf("Command %q exited with non-zero exit status: %d", cmd.Command, cmd.ExitStatus)
	}

	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh
	return err
}
