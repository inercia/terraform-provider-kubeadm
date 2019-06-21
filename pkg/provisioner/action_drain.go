package provisioner

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

// doDrainNode drains a node
func doDrainNode(d *schema.ResourceData) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("cannot not load Helm: no 'config_path' has been specified")
	}

	var node *v1.Node

	// TODO: get the Node.Name from the IP

	// Drain node (shelling out, FIXME after https://github.com/kubernetes/kubernetes/pull/72827 can be used [1.14])
	args := []string{"drain", "--delete-local-data=true", "--force=true", "--ignore-daemonsets=true", node.ObjectMeta.Name}
	return ssh.DoLocalKubectl(kubeconfig, args...)
}

func GetAdminClientSet(kubeconfig string) (*clientset.Clientset, error) {
	client, err := kubeconfigutil.ClientSetFromFile(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not load admin kubeconfig file")
	}
	return client, nil
}

func getMasterNodes(kubeconfig string) (*v1.NodeList, error) {
	clientSet, err := GetAdminClientSet(kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get admin clinet set")
	}

	return clientSet.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=", kubeadmconstants.LabelNodeRoleMaster),
	})
}

func isMaster(node *v1.Node) bool {
	_, isMaster := node.ObjectMeta.Labels[kubeadmconstants.LabelNodeRoleMaster]
	return isMaster
}
