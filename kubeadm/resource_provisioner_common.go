package kubeadm

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	// "k8s.io/client-go/kubernetes"
	// "k8s.io/client-go/tools/clientcmd"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
)

func doCommonProvisioning() ssh.ApplyFunc {
	return ssh.Composite(
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadFile(strings.NewReader(kubeletSysconfigCode), "/etc/sysconfig/kubelet"),
		ssh.DoUploadFile(strings.NewReader(kubeletServiceCode), "/usr/lib/systemd/system/kubelet.service"),
		ssh.DoUploadFile(strings.NewReader(kubeadmDropinCode), defKubeadmDropinPath),
	)
}

func doPrepareCRI() ssh.ApplyFunc {
	return ssh.Composite(
		ssh.DoUploadFile(strings.NewReader(CniDefConfCode), defCniLookbackConfPath),
		// we must reload the containers runtime engine after changing the CNI configuration
		ssh.ActionIf(
			ssh.CheckServiceExists("crio.service"),
			ssh.DoRestartService("crio.service")),
		ssh.ActionIf(
			ssh.CheckServiceExists("docker.service"),
			ssh.DoRestartService("docker.service")),
	)
}

// ///////////////////////////////////////////////////////////////////////////////////////

func getKubeadmIgnoredChecksArg(d *schema.ResourceData) string {
	ignoredChecks := defIgnorePreflightChecks[:]
	if checksOpt, ok := d.GetOk("ignore_checks"); ok {
		ignoredChecks = checksOpt.([]string)
	}

	if len(ignoredChecks) > 0 {
		return fmt.Sprintf("--ignore-preflight-errors=%s", strings.Join(ignoredChecks, ","))
	}

	return ""
}

func getKubeadmNodenameArg(d *schema.ResourceData) string {
	if nodenameOpt, ok := d.GetOk("nodename"); ok {
		return fmt.Sprintf("--node-name=%s", nodenameOpt.(string))
	}
	return ""
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
