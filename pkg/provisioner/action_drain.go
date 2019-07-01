package provisioner

import (
	"github.com/hashicorp/terraform/helper/schema"
	v1 "k8s.io/api/core/v1"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doDrainNode drains a node
func doDrainNode(d *schema.ResourceData) ssh.ApplyFunc {
	var node *v1.Node

	// TODO: get the Node.Name from the IP

	// Drain node (shelling out, FIXME after https://github.com/kubernetes/kubernetes/pull/72827 can be used [1.14])
	args := []string{"drain", "--delete-local-data=true", "--force=true", "--ignore-daemonsets=true", node.ObjectMeta.Name}
	return doLocalKubectl(d, args...)
}
