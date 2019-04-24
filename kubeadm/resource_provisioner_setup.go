package kubeadm

//go:generate ../utils/generate.sh --out-var kubeadmSetupScriptCode --out-package kubeadm --out-file generated_setup.go ./assets/kubeadm-setup.sh

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	defaultKubeadmSetup = "kubeadm-setup"
)

// doKubeadmSetup tries to install kubeadm in the remote machine
func doKubeadmSetup(o terraform.UIOutput, comm communicator.Communicator, installScript string, useSudo bool) error {

	var contents io.Reader

	// setup kubeadm
	if len(installScript) > 0 {
		o.Output(fmt.Sprintf("Uploading custom kubeadm installation script from %s...", installScript))
		f, err := os.Open(installScript)
		if err != nil {
			return err
		}
		contents = f
	} else {
		o.Output("Uploading built-in kubeadm installation script...")
		contents = strings.NewReader(kubeadmSetupScriptCode)
	}

	o.Output("Running kubeadm installation script")
	return ssh.DoExecScript(contents, defaultKubeadmSetup).Apply(o, comm, useSudo)
}
