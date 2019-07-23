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
	"bytes"
	"context"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

const (
	// command for getting the machine-id
	machineIDCmd = `cat /etc/machine-id`

	// command for getting a map of "machine-id <-> nodename"
	kubectlGetNodenameCmd = `get nodes -o yaml -o=jsonpath='{range .items[*]}{.status.nodeInfo.machineID}{"\t"}{.metadata.name}{"\n"}{end}'`
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

// DoGetNodename tries to get the nodename
func DoGetNodename(d *schema.ResourceData, node *ssh.KubeNode) ssh.Action {
	// maybe we can get it just from the `ResourceData`
	nodename := getNodenameFromResourceData(d)
	if len(nodename) > 0 {
		ssh.Debug("got nodename %q from resource data", node.Nodename)
		node.Nodename = nodename
		return nil
	}

	kubectl := getKubectlFromResourceData(d)
	kubeconfig := getKubeconfigFromResourceData(d)

	// otherwise, access the remote host
	return ssh.ActionFunc(func(ctx context.Context) ssh.Action {
		// first, get the machine ID
		ssh.Debug("trying to get the machine ID...")
		var buf bytes.Buffer
		res := ssh.DoSendingExecOutputToWriter(ssh.DoExec(machineIDCmd), &buf).Apply(ctx)
		if ssh.IsError(res) {
			return res
		}
		ssh.Debug("... output: %q", buf.String())
		machineID := strings.TrimSpace(buf.String())
		ssh.Debug("... machineID: %q", machineID)

		res = ssh.DoSendingExecOutputToFunc(
			ssh.DoRemoteKubectl(kubectl, kubeconfig, kubectlGetNodenameCmd),
			func(s string) {
				if len(s) == 0 {
					return
				}
				ssh.Debug("trying to find nodename in %q", s)
				if strings.Contains(s, machineID) {
					// parse:
					// bf38f8ac633e4f64a4924b0ed7b25946        kubeadm-master-0
					fields := strings.Fields(s)
					if len(fields) < 2 {
						ssh.Debug("could not get the nodename from fields: %+v", fields)
						return
					}
					node.Nodename = strings.TrimSpace(fields[1])
					ssh.Debug("... detected nodename %q", node.Nodename)
				}
			}).Apply(ctx)

		return res
	})
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
