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
	v1 "k8s.io/api/core/v1"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doDrainNode drains a node
func doDrainNode(d *schema.ResourceData) ssh.Action {
	var node *v1.Node

	// TODO: get the Node.Name from the IP

	// Drain node (shelling out, FIXME after https://github.com/kubernetes/kubernetes/pull/72827 can be used [1.14])
	args := []string{"drain", "--delete-local-data=true", "--force=true", "--ignore-daemonsets=true", node.ObjectMeta.Name}
	return doRemoteKubectl(d, args...)
}
