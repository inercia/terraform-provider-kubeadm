package kubeadm

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-kubeadm/internal/ssh"
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
