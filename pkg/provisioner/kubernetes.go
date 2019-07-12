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
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doRemoteKubectl runs a remote kubectl with the kubeconfig specified in the schema
func doRemoteKubectl(d *schema.ResourceData, args ...string) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	kubectl := getKubectlFromResourceData(d)
	return ssh.DoRemoteKubectl(kubectl, kubeconfig, args...)
}

// DoRemoteKubectlApply applies some manifests with a remote kubectl, uploading the kubeconfig specified in the schema
func doRemoteKubectlApply(d *schema.ResourceData, manifests []ssh.Manifest) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError("no 'config_path' has been specified")
	}
	return ssh.DoRemoteKubectlApply(getKubectlFromResourceData(d), kubeconfig, manifests)
}

// checkLocalKubeconfigAlive checks if a local kubeconfig exists and is alive
func checkLocalKubeconfigAlive(d *schema.ResourceData) ssh.CheckerFunc {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.CheckAnd(
		ssh.CheckLocalFileExists(getKubeconfigFromResourceData(d)),
		ssh.CheckAction(ssh.DoRemoteKubectl(getKubectlFromResourceData(d), kubeconfig, "cluster-info")))
}

// checkAdminConfAlive checks if a remmote kubeconfig exists and is alive
func checkAdminConfAlive(d *schema.ResourceData) ssh.CheckerFunc {
	return ssh.CheckAnd(
		ssh.CheckFileExists(ssh.DefAdminKubeconfig),
		// note: we will not use any "kubeconfig", so if "admin.conf" is not there it will just fail
		ssh.CheckAction(ssh.DoRemoteKubectl(getKubectlFromResourceData(d), "", "cluster-info")))
}

// doKubectlDrainNode runs a kubectl for draining a node
func doKubectlDrainNode(d *schema.ResourceData, nodename string) ssh.Action {
	args := []string{"drain",
		"--delete-local-data=true", "--force=true", "--ignore-daemonsets=true",
		nodename}

	ssh.Debug("running 'kubectl drain' command for %q", nodename)
	return ssh.ActionList{
		ssh.DoMessageInfo("Draining kubernetes node %q", nodename),
		doRemoteKubectl(d, args...),
	}
}

// doKubectlDeleteNode deletes the node from the cluster (so it will be forgotten forever)
func doKubectlDeleteNode(d *schema.ResourceData, nodename string) ssh.Action {
	args := []string{"delete", "node", nodename}

	ssh.Debug("running 'kubectl delete node' command for %q", nodename)
	return ssh.ActionList{
		ssh.DoMessageInfo("Deleting kubernetes node %q", nodename),
		doRemoteKubectl(d, args...),
	}
}

// doPrintKubeNodesSet prints the list of <nodename>:<IP> in the cluster
func doPrintKubeNodesSet(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError("no 'config_path' has been specified")
	}

	nodes := ssh.KubeNodesSet{}
	return ssh.DoTry(
		ssh.ActionList{
			ssh.DoGetKubeNodesSet(getKubectlFromResourceData(d), kubeconfig, &nodes),
			ssh.DoMessageInfo("Gathering Kubernetes nodes (and IPs) in the cluster..."),
			ssh.ActionFunc(func(ssh.Config) ssh.Action {
				if len(nodes) == 0 {
					return ssh.DoMessageWarn("no Kubernetes nodes detected.")
				}

				res := ssh.ActionList{}
				for ip, node := range nodes {
					res = append(res, ssh.DoMessageInfo("- ip:%s nodename:%s", ip, node.Nodename))
				}
				return res
			})})
}

// doDeleteLocalKubeconfig deletes the current, local kubeconfig (the one specified
// in the "config_path" attribute), but doing a backup first.
func doDeleteLocalKubeconfig(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	kubeconfigBak := kubeconfig + ".bak"

	return ssh.DoIf(
		ssh.CheckLocalFileExists(kubeconfig),
		ssh.ActionList{
			ssh.DoMessage("Removing local kubeconfig (with backup)"),
			ssh.DoMoveLocalFile(kubeconfig, kubeconfigBak),
		},
	)
}

// doDownloadKubeconfig downloads the "admin.conf" from the remote master
// to the local file specified in the "config_path" attribute
func doDownloadKubeconfig(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.DoDownloadFile(ssh.DefAdminKubeconfig, kubeconfig)
}

// doCheckLocalKubeconfigIsAlive checks that the local "kubeconfig" can be
// used for accessing the API server. In case we cannot, we just print
// a warning, as maybe the API server is not accessible from the localhost
// where Terraform is being run.
func doCheckLocalKubeconfigIsAlive(d *schema.ResourceData) ssh.Action {
	return ssh.ActionList{
		ssh.DoMessageInfo("Checking API server health and reachability..."),
		ssh.DoIfElse(
			checkLocalKubeconfigAlive(d),
			ssh.DoMessageInfo("The API server seems to be accessible from here."),
			ssh.DoMessageWarn("the API server does NOT seem to be accessible from here."),
		),
	}
}
