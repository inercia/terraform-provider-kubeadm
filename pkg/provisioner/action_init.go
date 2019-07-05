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

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doKubeadmInit runs the `kubeadm init`
func doKubeadmInit(d *schema.ResourceData) ssh.Applyer {
	_, initConfigBytes, err := common.InitConfigFromResourceData(d)
	if err != nil {
		return ssh.ApplyError(fmt.Sprintf("could not get a valid 'config' for init'ing: %s", err))
	}
	extraArgs := []string{"--skip-token-print"}

	actions := []ssh.Applyer{
		ssh.DoMessageInfo("Initializing the cluster with 'kubadm init'"),
		ssh.DoPrintIpAddresses(),
		doDeleteLocalKubeconfig(d),
		doUploadCerts(d),
		ssh.DoIfElse(
			ssh.CheckFileExists(ssh.DefAdminKubeconfig),
			ssh.DoMessage("admin.conf already exists: skipping `kubeadm init`"),
			doKubeadm(d, "init", initConfigBytes, extraArgs...),
		),
		doDownloadKubeconfig(d),
		doCheckKubeconfigIsAlive(d),
		doPrintEtcdMembers(d),
		doPrintNodes(d),
		doLoadCNI(d),
		doLoadDashboard(d),
		doLoadHelm(d),
		doLoadManifests(d),
	}

	return ssh.DoComposed(actions...)
}
