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
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// doLoadCNI loads the CNI driver
func doLoadCNI(d *schema.ResourceData) ssh.Applyer {
	manifest := ssh.Manifest{}
	message := ssh.DoNothing()

	if cniPluginManifestOpt, ok := d.GetOk("config.cni_plugin_manifest"); ok {
		cniPluginManifest := strings.TrimSpace(cniPluginManifestOpt.(string))
		if len(cniPluginManifest) > 0 {
			manifest = ssh.NewManifest(cniPluginManifest)
			if manifest.Inline != "" {
				return ssh.ApplyError(fmt.Sprintf("%q not recognized as URL or local filename", cniPluginManifest))
			}
			message = ssh.DoMessageInfo(fmt.Sprintf("Loading CNI plugin from %q", cniPluginManifest))
		}
	} else {
		if cniPluginOpt, ok := d.GetOk("config.cni_plugin"); ok {
			cniPlugin := strings.TrimSpace(strings.ToLower(cniPluginOpt.(string)))
			if len(cniPlugin) > 0 {
				log.Printf("[DEBUG] [KUBEADM] verifying CNI plugin: %s", cniPlugin)
				if template, ok := common.CNIPluginsManifestsTemplates[cniPlugin]; ok {
					log.Printf("[DEBUG] [KUBEADM] CNI plugin: %s", cniPlugin)
					config := d.Get("config").(map[string]interface{})
					replaced, err := common.ReplaceInTemplate(template, config)
					if err != nil {
						return ssh.ApplyError(fmt.Sprintf("could not replace variables in manifest for %q: %s", cniPlugin, err))
					}
					manifest.Inline = replaced
				} else {
					panic("unknown CNI driver: should have been caught at the validation stage")
				}
				message = ssh.DoMessageInfo(fmt.Sprintf("Loading CNI plugin %q", cniPlugin))
			}
		}
	}

	if manifest.IsEmpty() {
		return ssh.DoMessageWarn("no CNI driver is going to be loaded")
	}

	return ssh.DoComposed(
		message,
		doRemoteKubectlApply(d, []ssh.Manifest{manifest}))
}
