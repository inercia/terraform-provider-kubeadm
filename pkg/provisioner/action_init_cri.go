package provisioner

import (
	"strings"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doPrepareCRI preparse the CRI in the target node
func doPrepareCRI() ssh.ApplyFunc {
	return ssh.ApplyComposed(
		ssh.DoUploadFile(strings.NewReader(assets.CNIDefConfCode), common.DefCniLookbackConfPath),
		// we must reload the containers runtime engine after changing the CNI configuration
		ssh.ApplyIf(
			ssh.CheckServiceExists("crio.service"),
			ssh.DoRestartService("crio.service")),
		ssh.ApplyIf(
			ssh.CheckServiceExists("docker.service"),
			ssh.DoRestartService("docker.service")),
	)
}
