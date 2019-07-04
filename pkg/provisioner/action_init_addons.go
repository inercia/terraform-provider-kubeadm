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
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doLoadHelm loads Helm (if enabled)
func doLoadHelm(d *schema.ResourceData) ssh.Applyer {
	opt, ok := d.GetOk("config.helm_enabled")
	if !ok {
		return ssh.DoMessage("Helm will not be loaded")
	}
	enabled, err := strconv.ParseBool(opt.(string))
	if err != nil {
		panic("couold not parse helm_enabled in provisioner")
	}
	if !enabled {
		return ssh.DoMessage("Helm will not be loaded")
	}
	if common.DefHelmManifest == "" {
		return ssh.DoMessage("no manifest for Helm: Helm will not be loaded")
	}
	return doRemoteKubectlApply(d, []string{common.DefHelmManifest})
}

// doLoadDashboard loads the dashboard (if enabled)
func doLoadDashboard(d *schema.ResourceData) ssh.Applyer {
	opt, ok := d.GetOk("config.dashboard_enabled")
	if !ok {
		return ssh.DoMessage("the Dashboard will not be loaded")
	}
	enabled, err := strconv.ParseBool(opt.(string))
	if err != nil {
		panic("could not parse dashboard_enabled in provisioner")
	}
	if !enabled {
		return ssh.DoMessage("the Dashboard will not be loaded")
	}
	if common.DefDashboardManifest == "" {
		return ssh.DoMessage("no manifest for Dashboard: the Dashboard will not be loaded")
	}
	return doRemoteKubectlApply(d, []string{common.DefDashboardManifest})
}

// doLoadManifests loads some extra manifests
func doLoadManifests(d *schema.ResourceData) ssh.Applyer {
	manifestsOpt, ok := d.GetOk("manifests")
	if !ok {
		return ssh.DoNothing()
	}
	manifests := []string{}
	for _, v := range manifestsOpt.([]interface{}) {
		manifests = append(manifests, v.(string))
	}
	return doRemoteKubectlApply(d, manifests)
}
