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
	localKubeNode := ssh.KubeNode{}

	actions := ssh.ActionList{
		ssh.DoMessageInfo("Checking if we must drain the node from the Kubernetes cluster..."),
		DoGetNodename(d, &localKubeNode),
		ssh.ActionFunc(func(cfg ssh.Config) ssh.Action {
			if localKubeNode.IsEmpty() {
				return ssh.DoMessageWarn("could not find Kubernetes nodename for this node")
			}
			// drain the node with "nodename"
			return ssh.ActionList{
				doKubectlDrainNode(d, localKubeNode.Nodename),
				ssh.DoMessageInfo("Kubernetes node %q has been drained", localKubeNode.Nodename),
				doKubectlDeleteNode(d, localKubeNode.Nodename),
				ssh.DoMessageInfo("Kubernetes node %q has been deleted", localKubeNode.Nodename),
			}
		}),
	}
	return actions
}
