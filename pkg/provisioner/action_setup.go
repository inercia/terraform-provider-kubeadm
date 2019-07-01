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

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	defaultKubeadmSetup = "kubeadm-setup"
)

// doKubeadmSetup tries to install kubeadm in the remote machine
func doKubeadmSetup(d *schema.ResourceData, o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	if _, ok := d.GetOk("install"); ok {
		code := ""
		descr := ""
		auto := d.Get("install.0.auto").(bool)
		inline := d.Get("install.0.inline").(string)
		script := d.Get("install.0.script").(string)

		if auto {
			descr = "Uploading built-in kubeadm installation script..."
			code = assets.KubeadmSetupScriptCode
		} else if len(inline) > 0 {
			descr = "Uploading inlined installation script..."
			code = "#!/bin/sh\n" + inline
		} else if len(script) > 0 {
			descr = fmt.Sprintf("Uploading custom kubeadm installation script from %s...", script)
			contents, err := ioutil.ReadFile(script)
			if err != nil {
				o.Output(fmt.Sprintf("Error when reading installation script %q", script))
				return err
			}
			code = string(contents)
		}

		return ssh.DoComposed(
			ssh.DoMessage(descr),
			ssh.DoExecScript(strings.NewReader(code), defaultKubeadmSetup),
		).Apply(o, comm, useSudo)
	}
	return nil
}
