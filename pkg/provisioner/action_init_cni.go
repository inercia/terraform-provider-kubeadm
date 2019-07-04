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
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doLoadCNI loads the CNI driver
func doLoadCNI(d *schema.ResourceData) ssh.Applyer {
	manifest := ""
	if cniPluginManifestOpt, ok := d.GetOk("config.cni_plugin_manifest"); ok {
		cniPluginManifest := strings.TrimSpace(cniPluginManifestOpt.(string))
		if len(cniPluginManifest) > 0 {
			manifest = cniPluginManifest
		}
	} else {
		if cniPluginOpt, ok := d.GetOk("config.cni_plugin"); ok {
			cniPlugin := strings.TrimSpace(strings.ToLower(cniPluginOpt.(string)))
			if len(cniPlugin) > 0 {
				log.Printf("[DEBUG] [KUBEADM] verifying CNI plugin: %s", cniPlugin)
				if m, ok := common.CNIPluginsManifests[cniPlugin]; ok {
					log.Printf("[DEBUG] [KUBEADM] CNI plugin: %s", cniPlugin)
					manifest = m
				} else {
					panic("unknown CNI driver: should have been caught at the validation stage")
				}
			}
		}
	}

	if len(manifest) == 0 {
		return ssh.DoMessage("no CNI driver is going to be loaded")
	}
	return doRemoteKubectlApply(d, []string{manifest})
}
