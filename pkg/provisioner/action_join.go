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
	// get the join configuration
	joinConfig, _, err := common.JoinConfigFromResourceData(d)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// ... update the nodename
	joinConfig.NodeRegistration.Name = getNodenameFromResourceData(d)

	// ... and update the `config.join` section
	if err := common.JoinConfigToResourceData(d, joinConfig); err != nil {
		return ssh.ActionError(err.Error())
	}

	actions := ssh.ActionList{
		ssh.DoMessageInfo("Checking we have the required binaries..."),
		doCheckCommonBinaries(d),
		doRefreshToken(d),
		ssh.DoMessageInfo("Joining the cluster as a worker with 'kubadm join'..."),
		doKubeadm(d, "join"),
		doCheckLocalKubeconfigIsAlive(d),
		ssh.DoPrintIpAddresses(),
		doPrintEtcdMembers(d),
		doPrintNodes(d),
	}
	return actions
}

// doKubeadmJoinControlPlane runs the `kubeadm join` for another control-plane machine
func doKubeadmJoinControlPlane(d *schema.ResourceData) ssh.Action {
	// get the joinConfiguration from the 'config.join' in the ResourceData
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

	// add a local Control-Plane section to the JoinConfiguration (that means a new master will be started here)
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

	joinConfig.NodeRegistration.Name = getNodenameFromResourceData(d)

	// ... and update the `config.join` section in the ResourceData
	if err := common.JoinConfigToResourceData(d, joinConfig); err != nil {
		return ssh.ActionError(err.Error())
	}

	extraArgs := []string{}
	actions := ssh.ActionList{
		ssh.DoMessageInfo("Checking we have the required binaries..."),
		doCheckCommonBinaries(d),
		doRefreshToken(d),
		ssh.DoMessageInfo("Joining the cluster control-plane with 'kubadm join'..."),
		doUploadCerts(d),
		doKubeadm(d, "join", extraArgs...),
		doCheckLocalKubeconfigIsAlive(d),
		ssh.DoPrintIpAddresses(),
		doPrintEtcdMembers(d),
		doPrintNodes(d),
	}
	return actions
}
