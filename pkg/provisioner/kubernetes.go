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
	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doLocalKubectl runs a local kubectl with the kubeconfig specified in the schema
func doLocalKubectl(d *schema.ResourceData, args ...string) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.DoLocalKubectl(kubeconfig, args...)
}

// doRemoteKubectl runs a remote kubectl with the kubeconfig specified in the schema
func doRemoteKubectl(d *schema.ResourceData, args ...string) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.DoLocalKubectl(kubeconfig, args...)
}

// DoLocalKubectlApply applies some manifests with a local kubectl with the kubeconfig specified in the schema
func doLocalKubectlApply(d *schema.ResourceData, manifests []ssh.Manifest) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError("no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, manifests)
}

// DoRemoteKubectlApply applies some manifests with a remote kubectl, uploading the kubeconfig specified in the schema
func doRemoteKubectlApply(d *schema.ResourceData, manifests []ssh.Manifest) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError("no 'config_path' has been specified")
	}
	return ssh.DoRemoteKubectlApply(getKubectlFromResourceData(d), kubeconfig, manifests)
}

// doRefreshToken uses the kubeconfig for connecting to the API server and refreshing the token
func doRefreshToken(d *schema.ResourceData) ssh.Action {
	//token, ok := d.GetOk("config.token")
	//if !ok {
	//	panic("there should be a token")
	//}

	return ssh.DoIfElse(
		checkLocalKubeconfigAlive(d),
		ssh.ActionFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) ssh.Action {
			// TODO: we should (re)create the token by ssh'ing and doing a 'kubeadm token create'
			return nil
		}),
		ssh.DoAbort("no valid kubeconfig exists or the cluster is not alive/reachable: the token not refreshed, so the node cannot join the cluster"),
	)
}

// checkLocalKubeconfigExists checks if the kubeconfig exists
func checkLocalKubeconfigExists(d *schema.ResourceData) ssh.CheckerFunc {
	return ssh.CheckLocalFileExists(getKubeconfigFromResourceData(d))
}

// checkLocalKubeconfigAlive checks if the kubeconfig exists and is alive
func checkLocalKubeconfigAlive(d *schema.ResourceData) ssh.CheckerFunc {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.CheckAnd(
		checkLocalKubeconfigExists(d),
		ssh.CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
			if res := ssh.DoRemoteKubectl(getKubectlFromResourceData(d), kubeconfig, "cluster-info").Apply(o, comm, useSudo); ssh.IsError(res) {
				return false, nil // if some error happens, just return "false"
			}
			return true, nil
		}))
}

// checkAdminConfAlive checks if a remmote kubeconfig exists and is alive
func checkAdminConfAlive(d *schema.ResourceData) ssh.CheckerFunc {
	return ssh.CheckAnd(
		ssh.CheckFileExists(ssh.DefAdminKubeconfig),
		ssh.CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
			// note: we will not use any "kubeconfig", so if "admin.conf" is not there it will just fail
			if res := ssh.DoRemoteKubectl(getKubectlFromResourceData(d), "", "cluster-info").Apply(o, comm, useSudo); ssh.IsError(res) {
				return false, nil // if some error happens, just return "false"
			}
			return true, nil
		}))
}
