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
	"bytes"
	"fmt"
	"log"
	"os"
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
		ignoredChecks = append(ignoredChecks, checksOpt.([]string)...)
	}
	ignoredChecks = common.StringSliceUnique(ignoredChecks) // remove all the duplicates

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
	kubeadm_path := d.Get("install.0.kubeadm_path").(string)
	if len(kubeadm_path) == 0 {
		kubeadm_path = common.DefKubeadmPath
	}

	allArgs := []string{}
	switch command {
	case "init", "join":
		allArgs = append(allArgs, getKubeadmIgnoredChecksArg(d))
		allArgs = append(allArgs, getKubeadmNodenameArg(d))
		allArgs = append(allArgs, fmt.Sprintf("--config=%s", cfg))
	}

	// increase kubeadm verbosity if we are debugging at the Terraform level
	if _, ok := os.LookupEnv("TF_LOG"); ok {
		allArgs = append(allArgs, "-v3")
	}

	allArgs = append(allArgs, args...)
	return ssh.DoExec(fmt.Sprintf("%s %s %s", kubeadm_path, command, strings.Join(allArgs, " ")))
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

	// NOTE: the "install" block is optional, so there will be no
	// default values for "install.0.XXX" if the "install" block has not been given...
	sysconfigPath := d.Get("install.0.sysconfig_path").(string)
	if len(sysconfigPath) == 0 {
		sysconfigPath = common.DefKubeletSysconfigPath
	}

	servicePath := d.Get("install.0.service_path").(string)
	if len(servicePath) == 0 {
		servicePath = common.DefKubeletServicePath
	}

	dropinPath := d.Get("install.0.dropin_path").(string)
	if len(dropinPath) == 0 {
		dropinPath = common.DefKubeadmDropinPath
	}

	return ssh.DoComposed(
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletSysconfigCode), sysconfigPath),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletServiceCode), servicePath),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeadmDropinCode), dropinPath),
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
	if err := certsConfig.FromResourceDataConfig(d); err != nil {
		return ssh.DoAbort("no certificates data in config")
	}

	certsDir := common.DefPKIDir
	certsDirRaw, ok := d.GetOk("config.certs_dir")
	if ok {
		certsDir = certsDirRaw.(string)
	}

	actions := []ssh.Applyer{}
	for baseName, cert := range certsConfig.DistributionMap() {
		fullPath := path.Join(certsDir, baseName)
		log.Printf("[DEBUG] [KUBEADM] will upload certificate to %q", fullPath)
		upload := ssh.DoUploadReaderToFile(strings.NewReader(*cert), fullPath)
		actions = append(actions, upload)
	}

	return ssh.DoComposed(actions...)
}
