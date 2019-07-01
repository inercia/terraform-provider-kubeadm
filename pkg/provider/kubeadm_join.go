package provider

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// dataSourceToJoinConfig copies some settings to a Join configuration
func dataSourceToJoinConfig(d *schema.ResourceData, token string) (*kubeadmapi.JoinConfiguration, error) {
	joinConfig := &kubeadmapi.JoinConfiguration{
		NodeRegistration: kubeadmapi.NodeRegistrationOptions{
			KubeletExtraArgs: common.DefKubeletSettings,
		},
		Discovery: kubeadmapi.Discovery{
			BootstrapToken: &kubeadmapi.BootstrapTokenDiscovery{
				Token:                    token,
				UnsafeSkipCAVerification: true,
			},
		},
	}

	if _, ok := d.GetOk("runtime.0"); ok {
		if runtimeEngineOpt, ok := d.GetOk("runtime.0.engine"); ok {
			if socket, ok := common.DefCriSocket[runtimeEngineOpt.(string)]; ok {
				log.Printf("[DEBUG] [KUBEADM] setting CRI socket '%s'", socket)
				joinConfig.NodeRegistration.KubeletExtraArgs["container-runtime-endpoint"] = fmt.Sprintf("unix://%s", socket)
				joinConfig.NodeRegistration.CRISocket = socket
			} else {
				return nil, fmt.Errorf("unknown runtime engine %s", runtimeEngineOpt.(string))
			}
		}

		if _, ok := d.GetOk("runtime.0.extra_args.0"); ok {
			if args, ok := d.GetOk("runtime.0.extra_args.0.kubelet"); ok {
				joinConfig.NodeRegistration.KubeletExtraArgs = args.(map[string]string)
			}
		}
	}

	return joinConfig, nil
}
