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
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doKubeadmInit runs the `kubeadm init`
func doKubeadmInit(d *schema.ResourceData) ssh.Action {
	extraArgs := []string{"--skip-token-print"}

	// get the join configuration
	initConfig, _, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
	}

	// ... update the nodename
	initConfig.NodeRegistration.Name = getNodenameFromResourceData(d)

	// ... and update the `config.join` section
	if err := common.InitConfigToResourceData(d, initConfig); err != nil {
		return ssh.ActionError(err.Error())
	}

	actions := ssh.ActionList{
		// * if a "admin.conf" is there and the cluster is alive, do nothing
		//   (just try to reload CNI, Helm and so)
		// * if a partial setup is detected (ie, cluster is not alive but some manifests are there...)
		//   try to reset the node
		// * in any other case, do a regular "kubeadm init"
		doDeleteLocalKubeconfig(d),
		ssh.DoIfElse(
			checkAdminConfAlive(d),
			ssh.ActionList{
				ssh.DoMessageInfo("There is a 'admin.conf' in this master pointing to a live cluster: skipping any setup"),
			},
			ssh.ActionList{
				ssh.DoRetry(
					ssh.Retry{Times: 3, Interval: 15 * time.Second},
					ssh.ActionList{
						doMaybeResetMaster(d, common.DefKubeadmInitConfPath),
						doUploadCerts(d), // (we must upload certs because a "kubeadm reset" wipes them...)
						ssh.DoMessageInfo("Initializing the cluster with 'kubadm init'..."),
						doKubeadm(d, common.DefKubeadmInitConfPath, "init", extraArgs...),
					},
				),
			},
		),
		// we always download the kubeconfig and try to do a "kubeactl apply -f" of manifests
		doDownloadKubeconfig(d),
		doLoadCNI(d),
		doLoadDashboard(d),
		doLoadHelm(d),
		doLoadCloudProviderManager(d),
		doLoadExtraManifests(d),
	}
	return actions
}

// doMaybeResetMaster maybe "reset"s the master with kubeadm if
// it is detected as "partially" setup:
// ie, /etc/kubernetes/kubeadm-*.conf exist AND /etc/kubernetes/manifests/* exist
func doMaybeResetMaster(d *schema.ResourceData, kubeadmConfigFilename string) ssh.Action {
	return ssh.DoIf(
		ssh.CheckOr(
			ssh.CheckFileExists(kubeadmConfigFilename),
			ssh.CheckFileExists("/etc/kubernetes/manifests/kube-apiserver.yaml"),
			ssh.CheckFileExists("/etc/kubernetes/manifests/kube-controller-manager.yaml"),
			ssh.CheckFileExists("/etc/kubernetes/manifests/kube-scheduler.yaml"),
		),
		ssh.ActionList{
			ssh.DoMessageWarn("previous kubeadm config file found: resetting node"),
			doExecKubeadmWithConfig(d, "reset", "", "--force"),
			ssh.DoDeleteFile(kubeadmConfigFilename),
		})
}
