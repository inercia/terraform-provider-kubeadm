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
	"path/filepath"
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

// getKubeconfigFromResourceData returns the kubeconfig parameter passed in the `config_path`
func getKubeconfigFromResourceData(d *schema.ResourceData) string {
	kubeconfigOpt, ok := d.GetOk("config.config_path")
	if !ok {
		return ""
	}
	f, err := filepath.Abs(kubeconfigOpt.(string))
	if err != nil {
		return ""
	}
	return f
}

// getKubeadmFromResourceData returns the kubeadm binary path from the config
func getKubeadmFromResourceData(d *schema.ResourceData) string {
	kubeadm_path := d.Get("install.0.kubeadm_path").(string)
	if len(kubeadm_path) == 0 {
		kubeadm_path = common.DefKubeadmPath
	}
	return kubeadm_path
}

// getToken returns the current token
func getToken(d *schema.ResourceData) string {
	config := d.Get("config").(map[string]interface{})
	t, ok := config["token"]
	if !ok {
		return ""
	}
	return t.(string)
}

// getKubectlFromResourceData returns the kubectl binary path from the config
func getKubectlFromResourceData(d *schema.ResourceData) string {
	kubectl_path := d.Get("install.0.kubectl_path").(string)
	if len(kubectl_path) == 0 {
		kubectl_path = common.DefKubectlPath
	}
	return kubectl_path
}

// doExecKubeadmWithConfig runs a `kubeadm` command in the remote host
// this functions creates a `kubeadm` executor using some default values for some arguments.
func doExecKubeadmWithConfig(d *schema.ResourceData, command string, cfg string, args ...string) ssh.Action {
	kubeadm_path := getKubeadmFromResourceData(d)

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

// doKubeadm is the common kubeadm call, both for the `init` as well as well as for the `join`.
func doKubeadm(d *schema.ResourceData, command string, args ...string) ssh.Action {
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

	actions := ssh.ActionList{
		doPrepareCRI(),
		ssh.DoEnableService("kubelet.service"),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletSysconfigCode), sysconfigPath),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeletServiceCode), servicePath),
		ssh.DoUploadReaderToFile(strings.NewReader(assets.KubeadmDropinCode), dropinPath),
		doMaybeReset(d, kubeadmConfigFilename),
		doUploadJoinConfig(d, command, kubeadmConfigFilename),
		ssh.DoMessageInfo("Starting kubeadm..."),
		ssh.DoWithException(
			doExecKubeadmWithConfig(d, command, kubeadmConfigFilename, args...),
			ssh.DoDeleteFile(kubeadmConfigFilename)), // if something goes wrong, delete the "kubeadm.conf" file
		ssh.DoMoveFile(kubeadmConfigFilename, kubeadmConfigFilename+".bak"), // otherwise, back it up
	}
	return actions
}

// doMaybeReset maybe "reset"s with kubeadm if /etc/kubernetes/kubeadm-* exists
func doMaybeReset(d *schema.ResourceData, kubeadmConfigFilename string) ssh.Action {
	return ssh.DoIf(
		ssh.CheckFileExists(kubeadmConfigFilename),
		ssh.ActionList{
			doExecKubeadmWithConfig(d, "reset", "", "--force"),
			ssh.DoDeleteFile(kubeadmConfigFilename),
		})
}

func doUploadJoinConfig(d *schema.ResourceData, command string, kubeadmConfigFilename string) ssh.Action {
	return ssh.DoLazy(func() ssh.Action {
		// we must delay the joinConfig retrieval as some other functions modify it until the very last moment...
		joinConfigBytes := []byte{}
		var err error
		switch command {
		case "init":
			_, joinConfigBytes, err = common.InitConfigFromResourceData(d)
			if err != nil {
				return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for init'ing: %s", err))
			}

		case "join":
			_, joinConfigBytes, err = common.JoinConfigFromResourceData(d)
			if err != nil {
				return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
			}
		}
		return ssh.DoUploadReaderToFile(bytes.NewReader(joinConfigBytes), kubeadmConfigFilename)
	})
}

// doUploadCerts upload the certificates from the serialized `d.config` to the remote machine
// we only do this on the control plane machines
func doUploadCerts(d *schema.ResourceData) ssh.Action {
	certsConfig := &common.CertsConfig{}
	if err := certsConfig.FromResourceDataConfig(d); err != nil {
		return ssh.ActionError("no certificates data in config")
	}

	certsDir := common.DefPKIDir
	certsDirRaw, ok := d.GetOk("config.certs_dir")
	if ok {
		certsDir = certsDirRaw.(string)
	}

	actions := ssh.ActionList{}
	for baseName, cert := range certsConfig.DistributionMap() {
		fullPath := path.Join(certsDir, baseName)
		log.Printf("[DEBUG] [KUBEADM] will upload certificate to %q", fullPath)
		upload := ssh.DoUploadReaderToFile(strings.NewReader(*cert), fullPath)
		actions = append(actions, upload)
	}

	return actions
}

// doPrintNodes prints the list of <nodename>:<IP> in the cluster
func doPrintNodes(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	if kubeconfig == "" {
		return ssh.ActionError("no 'config_path' has been specified")
	}

	ipAddresses := map[string]string{}
	return ssh.DoTry(
		ssh.ActionList{
			ssh.DoGetNodesAndIPs(getKubectlFromResourceData(d), kubeconfig, ipAddresses),
			ssh.DoMessage("Nodes (and IPs) in cluster:"),
			ssh.DoLazy(func() ssh.Action {
				res := ssh.ActionList{}
				for ip, name := range ipAddresses {
					res = append(res, ssh.DoMessage("- ip:%s name:%s", ip, name))
				}
				return res
			})})
}

// doCheckCommonBinaries checks that some common binaries neccessary are present in the remote machine
func doCheckCommonBinaries(d *schema.ResourceData) ssh.Action {
	checks := ssh.ActionList{}

	kubeadm_path := getKubeadmFromResourceData(d)
	checks = append(checks,
		ssh.DoIfElse(
			ssh.CheckBinaryExists(kubeadm_path),
			ssh.DoMessage("- kubeadm found"),
			ssh.DoAbort("kubeadm NOT found in $PATH. You can specify a custom executable in the 'install.kubeadm_path' property in the provisioner.")))

	kubectl_path := getKubectlFromResourceData(d)
	checks = append(checks,
		ssh.DoIfElse(
			ssh.CheckBinaryExists(kubectl_path),
			ssh.DoMessage("- kubectl found"),
			ssh.DoAbort("kubectl NOT found in $PATH. You can specify a custom executable in the 'install.kubectl_path' property in the provisioner")))

	return checks
}
