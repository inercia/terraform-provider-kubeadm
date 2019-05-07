package provisioner

import (
	"bytes"
	"errors"
	"fmt"
	"path"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

var (
	errNoJoinConfigFound = errors.New("no join configuration obtained")
)

// doKubeadmJoin performs a `kubeadm join` in the remote host
func doKubeadmJoin(d *schema.ResourceData, configFile []byte) ssh.ApplyFunc {
	kubeadmConfigFile := path.Join(common.DefKubeadmJoinConfPath)
	extraArgs := ""
	extraArgs += " " + getKubeadmIgnoredChecksArg(d)
	extraArgs += " " + getKubeadmNodenameArg(d)

	return ssh.ApplyComposed(
		ssh.ApplyIf(
			ssh.CheckFileExists(kubeadmConfigFile),
			ssh.DoExec("kubeadm reset --force")),
		ssh.DoUploadFile(bytes.NewReader(configFile), kubeadmConfigFile),
		ssh.DoExec(fmt.Sprintf("kubeadm join --config=%s %s", kubeadmConfigFile, extraArgs)),
	)
}
