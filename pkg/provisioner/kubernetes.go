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

// getClientset gets a clientset, or "nil" if no "kubeconfig" has been provided
// Usage:
//      clientset := getClientset(d)
// 		pods, _ := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
// 		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
//
// func getClientset(d *schema.ResourceData) (*kubernetes.Clientset, error) {
// 	if kubeconfigOpt, ok := d.GetOk("kubeconfig"); ok {
// 		kubeconfig := kubeconfigOpt.(string)

// 		config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
// 		return kubernetes.NewForConfig(config)
// 	}
// 	return nil, nil
// }

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

// doLocalKubectl runs a local kubectl with the kubeconfig specified in the schema
func doLocalKubectl(d *schema.ResourceData, args ...string) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectl(kubeconfig, args...)
}

// DoLocalKubectlApply applies some manifests with a local kubectl with the kubeconfig specified in the schema
func doLocalKubectlApply(d *schema.ResourceData, manifests []string) ssh.ApplyFunc {
	kubeconfig := getKubeconfig(d)
	if kubeconfig == "" {
		return ssh.DoAbort("no 'config_path' has been specified")
	}
	return ssh.DoLocalKubectlApply(kubeconfig, manifests)
}
