package provisioner

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

//
// kubeadm
//

// getKubeadmIgnoredChecksArg returns the kubeadm arguments for the ignored checks
func getKubeadmIgnoredChecksArg(d *schema.ResourceData) string {
	ignoredChecks := common.DefIgnorePreflightChecks[:]
	if checksOpt, ok := d.GetOk("ignore_checks"); ok {
		ignoredChecks = checksOpt.([]string)
	}

	if len(ignoredChecks) > 0 {
		return fmt.Sprintf("--ignore-preflight-errors=%s", strings.Join(ignoredChecks, ","))
	}

	return ""
}

// getKubeadmNodenameArg returns the kubeadm arguments for specifying the nodename
func getKubeadmNodenameArg(d *schema.ResourceData) string {
	if nodenameOpt, ok := d.GetOk("nodename"); ok {
		return fmt.Sprintf("--node-name=%s", nodenameOpt.(string))
	}
	return ""
}

// getKubeconfig returns the kubeconfig parameter passed in the `config_path`
func getKubeconfig(d *schema.ResourceData) string {
	kubeconfigOpt, ok := d.GetOk("config.config_path")
	if !ok {
		return ""
	}
	return kubeconfigOpt.(string)
}

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
