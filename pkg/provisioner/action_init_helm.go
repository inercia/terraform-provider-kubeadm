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
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/helm/cmd/helm/installer"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	defHelmReplicas  = 1
	defHelmNamespace = "kube-system"
	// defHelmNodeselector = "node-role.kubernetes.io/master="
	defHelmNodeselector = ""
)

// doLoadHelm loads Helm (if enabled)
func doLoadHelm(d *schema.ResourceData) ssh.Action {
	opt, ok := d.GetOk("config.helm_enabled")
	if !ok {
		return ssh.DoMessageWarn("Helm will not be loaded")
	}
	enabled, err := strconv.ParseBool(opt.(string))
	if err != nil {
		return ssh.ActionError("could not parse helm_enabled in provisioner")
	}
	if !enabled {
		return ssh.DoMessageWarn("Helm will not be loaded")
	}

	opts := installer.Options{
		Namespace:                    defHelmNamespace,
		AutoMountServiceAccountToken: true,
		EnableHostNetwork:            false,
		NodeSelectors:                defHelmNodeselector,
		UseCanary:                    false,
		Replicas:                     defHelmReplicas,
		// TODO: we shoud have options for enabling TLS and so...
	}
	manifests, err := installer.TillerManifests(&opts)
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not get Tiller manifests for installing Helm: %s", err))
	}

	allManifests := strings.Join(manifests, "\n---\n")
	ssh.Debug("Helm manifests: %s", allManifests)

	return ssh.ActionList{
		ssh.DoMessageInfo("Loading Helm..."),
		doRemoteKubectlApply(d, []ssh.Manifest{{Inline: allManifests}}),
	}
}
