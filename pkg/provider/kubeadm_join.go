// Copyright © 2019 Alvaro Saurin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
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
				ssh.Debug("setting CRI socket '%s'", socket)
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

	// check if we have some cloud-provider
	if cloudProvRaw, ok := d.GetOk("cloud.0.provider"); ok && len(cloudProvRaw.(string)) > 0 {
		joinConfig.NodeRegistration.KubeletExtraArgs["cloud-provider"] = "external"
	}

	return joinConfig, nil
}
