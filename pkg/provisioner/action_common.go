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

// expectedBinaries is the list of expected binaries to be present in the remote machine
var expectedBinaries = []struct {
	name        string
	defaultPath func(*schema.ResourceData) string
	property    string
}{
	{
		name:        "kubeadm",
		defaultPath: getKubeadmFromResourceData,
		property:    "install.kubeadm_path",
	},
	{
		name:        "kubectl",
		defaultPath: getKubectlFromResourceData,
		property:    "install.kubectl_path",
	},
}

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

// doExecKubeadmWithConfig runs a `kubeadm` command in the remote host
// this functions creates a `kubeadm` executor using some default values for some arguments.
func doExecKubeadmWithConfig(d *schema.ResourceData, command string, cfg string, args ...string) ssh.Action {
	kubeadm_path := getKubeadmFromResourceData(d)

	allArgs := []string{}
	switch command {
	case "init", "join":
		allArgs = append(allArgs, getKubeadmIgnoredChecksArg(d))
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
		// run kubeadm... if something goes wrong, delete the "kubeadm.conf" file created
		// otherwise, back up the config file
		ssh.DoMessageInfo("Starting kubeadm..."),
		ssh.DoWithException(
			ssh.ActionList{
				doUploadKubeadmConfig(d, command, kubeadmConfigFilename),
				doExecKubeadmWithConfig(d, command, kubeadmConfigFilename, args...),
			},
			ssh.DoDeleteFile(kubeadmConfigFilename)),
		ssh.DoMoveFile(kubeadmConfigFilename, kubeadmConfigFilename+".bak"),
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

func doUploadKubeadmConfig(d *schema.ResourceData, command string, kubeadmConfigFilename string) ssh.Action {
	return ssh.DoLazy(func() ssh.Action {
		// we must delay the {init|join}Config retrieval as some other functions
		// modify it until the very last moment...
		configBytes := []byte{}
		var err error
		switch command {
		case "init":
			_, configBytes, err = common.InitConfigFromResourceData(d)
			if err != nil {
				return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for init'ing: %s", err))
			}

		case "join":
			_, configBytes, err = common.JoinConfigFromResourceData(d)
			if err != nil {
				return ssh.ActionError(fmt.Sprintf("could not get a valid 'config' for join'ing: %s", err))
			}
		}
		return ssh.DoUploadReaderToFile(bytes.NewReader(configBytes), kubeadmConfigFilename)
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
	for _, expected := range expectedBinaries {
		path := expected.defaultPath(d)
		checks = append(checks,
			ssh.DoIfElse(
				ssh.CheckBinaryExists(path),
				ssh.DoMessage("- %s found", expected.name),
				ssh.DoAbort("%s NOT found in $PATH. You can specify a custom executable in the '%s' property in the provisioner.", expected.name, expected.property)))
	}

	return checks
}

// doDeleteLocalKubeconfig deletes the current, local kubeconfig (the one specified
// in the "config_path" attribute), but doing a backup first.
func doDeleteLocalKubeconfig(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	kubeconfigBak := kubeconfig + ".bak"

	return ssh.DoIf(
		ssh.CheckLocalFileExists(kubeconfig),
		ssh.ActionList{
			ssh.DoMessage("Removing local kubeconfig (with backup)"),
			ssh.DoMoveLocalFile(kubeconfig, kubeconfigBak),
		},
	)
}

// doDownloadKubeconfig downloads the "admin.conf" from the remote master
// to the local file specified in the "config_path" attribute
func doDownloadKubeconfig(d *schema.ResourceData) ssh.Action {
	kubeconfig := getKubeconfigFromResourceData(d)
	return ssh.DoDownloadFile(ssh.DefAdminKubeconfig, kubeconfig)
}

// doCheckLocalKubeconfigIsAlive checks that the local "kubeconfig" can be
// used for accessing the API server. In case we cannot, we just print
// a warning, as maybe the API server is not accessible from the localhost
// where Terraform is being run.
func doCheckLocalKubeconfigIsAlive(d *schema.ResourceData) ssh.Action {
	return ssh.DoIfElse(
		checkLocalKubeconfigAlive(d),
		ssh.DoMessageInfo("the API server is accessible from here (with the current kubeconfig)"),
		ssh.DoMessageWarn("the API server does NOT seem to be accessible from here (with the current kubeconfig)"),
	)
}
