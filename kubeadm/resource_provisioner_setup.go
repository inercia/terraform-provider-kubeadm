package kubeadm

//go:generate ../utils/generate.sh --out-var kubeadmSetupScriptCode --out-package kubeadm --out-file generated_setup.go ./assets/kubeadm-setup.sh

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

const (
	defaultKubeadmSetup = "kubeadm-setup"
)

// DoKubeadmSetup tries to install kubeadm in the remote machine
func DoKubeadmSetup(o terraform.UIOutput, comm communicator.Communicator, installScript string, useSudo bool) error {
	// setup kubeadm
	remoteSetupScript := newRemoteScript(o, comm)
	if len(installScript) > 0 {
		o.Output(fmt.Sprintf("Uploading custom kubeadm installation script from %s...", installScript))
		f, err := os.Open(installScript)
		if err != nil {
			return err
		}
		if err := remoteSetupScript.UploadScript(f, defaultKubeadmSetup); err != nil {
			return err
		}
	} else {
		o.Output("Uploading default kubeadm installation script...")
		if err := remoteSetupScript.UploadScript(strings.NewReader(kubeadmSetupScriptCode), defaultKubeadmSetup); err != nil {
			return err
		}
	}

	o.Output("Running kubeadm installation script")
	if err := remoteSetupScript.Run("", useSudo); err != nil {
		return err
	}

	return nil
}
