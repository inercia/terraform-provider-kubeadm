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

package provisioner

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doKubeadmJoinWorker runs the `kubeadm join`
func doKubeadmJoinWorker(d *schema.ResourceData) ssh.Action {
	_, joinConfigBytes, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// check if we are joining the Control Plane: we must upload the certificates and
	// use the '--control-plane' flag
	actions := ssh.ActionList{
		ssh.DoMessageInfo("Joining the cluster as a worker with 'kubadm join'"),
		ssh.DoPrintIpAddresses(),
		doKubeadm(d, "join", joinConfigBytes),
		doCheckKubeconfigIsAlive(d),
		doPrintEtcdMembers(d),
		doPrintNodes(d),
	}
	return actions
}

// doKubeadmJoinWorker runs the `kubeadm join`
func doKubeadmJoinControlPlane(d *schema.ResourceData) ssh.Action {
	joinConfig, _, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// check that we have a stable control plane endpoint
	initConfig, _, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}
	if len(initConfig.ClusterConfiguration.ControlPlaneEndpoint) == 0 {
		return ssh.ActionError("Cannot create additional masters when the 'kubeadm.<name>.api.external' is empty")
	}

	// add a local Control-Plane section to the JoinConfiguration
	endpoint := kubeadmapi.APIEndpoint{}
	if hp, ok := d.GetOk("listen"); ok {
		h, p, err := common.SplitHostPort(hp.(string), common.DefAPIServerPort)
		if err != nil {
			return ssh.ActionError(fmt.Sprintf("could not parse listen address %q: %s", hp.(string), err))
		}
		endpoint = kubeadmapi.APIEndpoint{AdvertiseAddress: h, BindPort: int32(p)}
	} else {
		endpoint = kubeadmapi.APIEndpoint{AdvertiseAddress: "", BindPort: common.DefAPIServerPort}
	}
	joinConfig.ControlPlane = &kubeadmapi.JoinControlPlane{LocalAPIEndpoint: endpoint}

	joinConfigBytes, err := common.JoinConfigToYAML(joinConfig)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	extraArgs := []string{}
	actions := ssh.ActionList{
		ssh.DoMessageInfo("Joining the cluster control-plane with 'kubadm join'"),
		ssh.DoPrintIpAddresses(),
		doUploadCerts(d),
		doKubeadm(d, "join", joinConfigBytes, extraArgs...),
		doCheckKubeconfigIsAlive(d),
		doPrintEtcdMembers(d),
		doPrintNodes(d),
	}
	return actions
}
