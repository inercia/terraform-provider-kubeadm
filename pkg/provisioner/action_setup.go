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
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doKubeadmSetup tries to install kubeadm in the remote machine
// the auto-installation can be
// 1) our built-in auto-installation script
// 2) a user-provided script in some path
// 3) an inlined user-provided script
func doKubeadmSetup(d *schema.ResourceData) ssh.Action {
	if _, ok := d.GetOk("install"); ok {
		code := ""
		descr := ""
		auto := d.Get("install.0.auto").(bool)
		inline := d.Get("install.0.inline").(string)
		script := d.Get("install.0.script").(string)

		if auto {
			ssh.Debug("will upload the builtin auto-installation script")
			descr = "Uploading and running built-in kubeadm installation script..."
			code = assets.KubeadmSetupScriptCode
		} else if len(inline) > 0 {
			ssh.Debug("will upload auto-installation script from inlined script: %d bytes", len(inline))
			descr = "Uploading and running inlined installation script..."
			code = "#!/bin/sh\n" + inline
		} else if len(script) > 0 {
			ssh.Debug("will upload auto-installation from custom script from %q", script)
			descr = fmt.Sprintf("Uploading and running custom kubeadm script from %s...", script)
			contents, err := ioutil.ReadFile(script)
			if err != nil {
				errMsg := fmt.Sprintf("when reading kubeadm setup script %q: %s", script, err.Error())
				return ssh.ActionError(errMsg)
			}
			code = string(contents)
		}

		return ssh.ActionList{
			ssh.DoMessage(descr),
			ssh.DoExecScript(strings.NewReader(code)),
		}
	}
	return ssh.ActionList{
		ssh.DoMessageWarn("no auto-installation: assuming kubeadm is installed in the target node."),
	}
}
