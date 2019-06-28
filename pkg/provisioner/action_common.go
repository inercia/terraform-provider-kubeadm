package provisioner

import (
	"bytes"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/assets"
	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

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

// doExecKubeadmWithConfig runs a `kubeadm` command in the remote host
// this functions creates a `kubeadm` executor using some default values for some arguments.
func doExecKubeadmWithConfig(d *schema.ResourceData, command string, cfg string, args ...string) ssh.ApplyFunc {
	allArgs := []string{}
	switch command {
	case "init", "join":
		allArgs = append(allArgs, getKubeadmIgnoredChecksArg(d))
		allArgs = append(allArgs, getKubeadmNodenameArg(d))
		allArgs = append(allArgs, fmt.Sprintf("--config=%s", cfg))
	}

	allArgs = append(allArgs, args...)
	return ssh.DoExec(fmt.Sprintf("kubeadm %s %s", command, strings.Join(allArgs, " ")))
}

// doKubeadm are the common provisioning things, for the `init` as well
// as for the `join`.
func doKubeadm(d *schema.ResourceData, command string, kubeadmConfig []byte, args ...string) ssh.ApplyFunc {
	kubeadmConfigFilename := ""
	switch command {
	case "init":
		kubeadmConfigFilename = common.DefKubeadmInitConfPath
	case "join":
		kubeadmConfigFilename = common.DefKubeadmJoinConfPath
	}

	return ssh.DoComposed(
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletSysconfigCode), "/etc/sysconfig/kubelet"),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletServiceCode), "/usr/lib/systemd/system/kubelet.service"),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeadmDropinCode), common.DefKubeadmDropinPath),
		ssh.DoComposed(
			ssh.DoIf(
				ssh.CheckFileExists(kubeadmConfigFilename),
				ssh.DoComposed(
					doExecKubeadmWithConfig(d, "reset", "", "--force"),
					ssh.DoDeleteFile(kubeadmConfigFilename),
				)),
			ssh.DoUploadReaderToFile(bytes.NewReader(kubeadmConfig), kubeadmConfigFilename),
			doExecKubeadmWithConfig(d, command, kubeadmConfigFilename, args...),
			ssh.DoMoveFile(kubeadmConfigFilename, kubeadmConfigFilename+".bak"),
		),
	)
}

// doUploadCerts upload the certificates from the serialized `d.config` to the remote machine
// we only do this on the control plane machines
func doUploadCerts(d *schema.ResourceData) ssh.ApplyFunc {
	certsConfig := &common.CertsConfig{}
	if err := certsConfig.FromResourceData(d); err != nil {
		return ssh.DoAbort("no certificates data in config")
	}

	actions := []ssh.Applyer{}
	for baseName, cert := range certsConfig.DistributionMap() {
		fullPath := path.Join(certsConfig.Dir, baseName)
		log.Printf("[DEBUG] [KUBEADM] will upload certificate %q", fullPath)
		upload := ssh.DoUploadReaderToFile(strings.NewReader(*cert), fullPath)
		actions = append(actions, upload)
	}

	return ssh.DoComposed(actions...)
}
