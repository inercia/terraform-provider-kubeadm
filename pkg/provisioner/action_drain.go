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

func doRemoveNode(d *schema.ResourceData) ssh.Action {
	return ssh.ActionList{
		ssh.DoMessageInfo("Preparing to remove node from cluster..."),
		ssh.DoTry(doDrainKubernetesNode(d)),
		ssh.DoTry(doRemoveIfMember(d)),
	}
}

// doDrainKubernetesNode drains a Kubernetes node
func doDrainKubernetesNode(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	kubectl := getKubectlFromResourceData(d)
	localIPs := []string{}
	nodes := ssh.KubeNodesSet{}

	actions := ssh.ActionList{
		ssh.DoMessageInfo("Checking if we must drain the node from the Kubernetes cluster..."),
		// get the list of local IP addresses
		ssh.DoGetIpAddresses(&localIPs),
		// get the map of nodes, as IP<->nodename, as known by the API server
		ssh.DoGetKubeNodesSet(kubectl, kubeconfig, &nodes),
		ssh.ActionFunc(func(cfg ssh.Config) ssh.Action {
			if len(localIPs) == 0 {
				return ssh.DoMessageWarn("no local IPs detected: cannot remove node with 'kubectl'")
			}

			// check if this is the last node in the cluster: in that case, don't do anything
			ssh.Debug("current nodes: %s", nodes)
			if len(nodes) == 0 {
				return ssh.DoMessageWarn("no Kubernetes nodes detected (maybe the API server is not working): will not spend time draining the node with 'kubectl'")
			}
			if len(nodes) == 1 {
				return ssh.DoMessageWarn("last node in the cluster: will not spend time draining the node with 'kubectl'")
			}

			// get the nodename for this host
			// we do that by going through all the nodenames known by the API server
			// trying to match the IP addresses we have for this host
			nodename := ""
			for _, localIP := range localIPs {
				ssh.Debug("checking if %q has some nodename", localIP)
				if kubenode, ok := nodes[localIP]; ok {
					// ok, we have a nodename for that IP
					ssh.Debug("obtained local nodename: %q", kubenode)
					nodename = kubenode.Nodename
				}
			}

			if nodename == "" {
				return ssh.DoMessageWarn("no nodename found in the API server for current host")
			}

			// drain the node with "nodename"
			return ssh.ActionList{
				doKubectlDrainNode(d, nodename),
				ssh.DoMessageInfo("Kubernetes node %q has been drained", nodename),
				doKubectlDeleteNode(d, nodename),
				ssh.DoMessageInfo("Kubernetes node %q has been deleted", nodename),
			}
		}),
	}
	return actions
}
