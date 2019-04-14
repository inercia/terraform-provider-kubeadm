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

func DoKubeadmInit(o terraform.UIOutput, comm communicator.Communicator,
	initConfig *kubeadm.InitConfiguration, configFile []byte, useSudo bool) error {
	configPath := path.Join("/etc/kubernetes/kubeadm-init.conf")

	replacements := struct {
		Phase             string
		InitConfiguration *kubeadm.InitConfiguration
	}{
		"init",
		initConfig,
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
			descr:    "kubeadm init",
			path:     configPath,
		},
	}
	if err := newRemoteTemplates(o, comm).Upload(templates, replacements); err != nil {
		return err
	}

	o.Output(fmt.Sprintf("Initializing the cluster with 'kubadm init' with %s", configPath))
	commands := []string{
		"kubeadm reset || /bin/true",
		"systemctl restart kubelet || /bin/true",
		fmt.Sprintf("kubeadm init --config=%s", configPath),
	}
	return newRemoteCommands(o, comm).Run(commands, useSudo)
}
