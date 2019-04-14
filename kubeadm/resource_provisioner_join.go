package kubeadm

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"

	kubeadm "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
)

func DoKubeadmJoin(o terraform.UIOutput, comm communicator.Communicator,
	joinConfig *kubeadm.JoinConfiguration, configFile []byte, useSudo bool) error {
	configPath := path.Join("/etc/kubernetes/kubeadm-join.conf")

	replacements := struct {
		Phase             string
		JoinConfiguration *kubeadm.JoinConfiguration
	}{
		"init",
		joinConfig,
	}

	templates := []remoteTemplate{
		{
			contents: strings.NewReader(kubeletSysconfigCode),
			descr:    "kubelet sysconfig",
			path:     "/etc/sysconfig/kubelet",
		},
		{
			contents: strings.NewReader(kubeletServiceCode),
			descr:    "kubelet service",
			path:     "/usr/lib/systemd/system/kubelet.service",
		},
		{
			contents: strings.NewReader(kubeadmDropinCode),
			descr:    "kubeadm dropin",
			path:     "/usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf",
		},
		{
			contents: bytes.NewReader(configFile),
			descr:    "kubeadm join (as a worker)",
			path:     configPath,
		},
	}
	if err := newRemoteTemplates(o, comm).Upload(templates, replacements); err != nil {
		return err
	}

	o.Output(fmt.Sprintf("joining the cluster with 'kubadm join' with %s", configPath))
	commands := []string{
		"kubeadm reset || /bin/true",
		"systemctl restart kubelet || /bin/true",
		fmt.Sprintf("kubeadm join --config=%s", configPath),
	}
	return newRemoteCommands(o, comm).Run(commands, useSudo)
}
