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
	"context"
	"fmt"
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
	//{
	//	name:        "hostname",
	//	defaultPath: nil,
	//	property:    "",
	//},
}

// getKubeadmIgnoredChecksArg returns the kubeadm arguments for the ignored checks
func getKubeadmIgnoredChecksArg(d *schema.ResourceData) string {
	ignoredChecks := common.DefIgnorePreflightChecks[:]
	if checksOptRaw, ok := d.GetOk("ignore_checks"); ok {
		checksOpts := checksOptRaw.([]interface{})
		for _, check := range checksOpts {
			ignoredChecks = append(ignoredChecks, check.(string))
		}
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
func doKubeadm(d *schema.ResourceData, kubeadmConfigFilename string, command string, args ...string) ssh.Action {
	// run kubeadm... if something goes wrong, delete the "kubeadm-*.conf" file created
	// otherwise, back up the config file
	actions := ssh.ActionList{
		ssh.DoMessageInfo("Starting kubeadm..."),
		ssh.DoWithException(ssh.ActionList{
			doUploadKubeadmConfig(d, command, kubeadmConfigFilename),
			doExecKubeadmWithConfig(d, command, kubeadmConfigFilename, args...),
		}, ssh.ActionList{
			ssh.DoTry(ssh.DoDeleteFile(kubeadmConfigFilename))}),
		ssh.DoTry(ssh.DoMoveFile(kubeadmConfigFilename, kubeadmConfigFilename+".bak")),
	}
	return actions
}

// doMaybeResetWorker maybe "reset"s with kubeadm if /etc/kubernetes/kubeadm-* exists
func doMaybeResetWorker(d *schema.ResourceData, kubeadmConfigFilename string) ssh.Action {
	return ssh.DoIf(
		ssh.CheckFileExists(kubeadmConfigFilename),
		ssh.ActionList{
			ssh.DoMessageWarn("previous kubeadm config file found: resetting node"),
			doExecKubeadmWithConfig(d, "reset", "", "--force"),
			ssh.DoDeleteFile(kubeadmConfigFilename),
		})
}

func doUploadKubeadmConfig(d *schema.ResourceData, command string, kubeadmConfigFilename string) ssh.Action {
	return ssh.ActionFunc(func(context.Context) ssh.Action {
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
		return ssh.DoUploadBytesToFile(configBytes, kubeadmConfigFilename)
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

	actions := ssh.ActionList{
		ssh.DoMessageInfo("Uploading certificates..."),
	}
	for baseName, cert := range certsConfig.DistributionMap() {
		fullPath := path.Join(certsDir, baseName)
		ssh.Debug("will upload certificate to %q", fullPath)
		upload := ssh.DoUploadBytesToFile([]byte(*cert), fullPath)
		actions = append(actions, upload)
	}

	return actions
}

// doLoadCloudProviderManager uploads the cloud-config to /etc/kubernetes/cloud.conf if necessary
func doLoadCloudProviderManager(d *schema.ResourceData) ssh.Action {
	cloudProviderRaw, ok := d.GetOk("config.cloud_provider")
	if !ok {
		return nil
	}

	cloudProvider := cloudProviderRaw.(string)
	if len(cloudProvider) == 0 {
		return nil
	}

	manifest := ssh.Manifest{Inline: assets.CloudProviderCode}
	err := manifest.ReplaceConfig(common.GetProvisionerConfig(d))
	if err != nil {
		return ssh.ActionError(fmt.Sprintf("could not replace variables in cloud controller manager manifest for %q: %s", cloudProvider, err))
	}
	actions := ssh.ActionList{
		ssh.DoMessageInfo("Loading cloud controller manager for %q", cloudProvider),
		doRemoteKubectlApply(d, []ssh.Manifest{manifest}),
	}
	return actions
}

// doCheckCommonBinaries checks that some common binaries neccessary are present in the remote machine
func doCheckCommonBinaries(d *schema.ResourceData) ssh.Action {

	checks := ssh.ActionList{}
	for _, expected := range expectedBinaries {
		path := expected.name
		if expected.defaultPath != nil {
			path = expected.defaultPath(d)
		}

		abortMsg := fmt.Sprintf("%s NOT found in $PATH.", expected.name)
		if expected.property != "" {
			abortMsg += fmt.Sprintf(" You can specify a custom executable in the '%s' property in the provisioner.", expected.property)
		}

		checks = append(checks,
			ssh.DoIfElse(
				ssh.CheckBinaryExists(path),
				ssh.DoMessageInfo("- %s found", expected.name),
				ssh.DoAbort(abortMsg)))
	}

	return checks
}

// doUploadResolvConf uploads some configuration for DNS upstream servers
func doUploadResolvConf(d *schema.ResourceData) ssh.Action {
	dRaw, ok := d.GetOk("config.dns_upstream")
	if !ok {
		return nil
	}

	upstreamStr := dRaw.(string)
	if len(upstreamStr) == 0 {
		return nil
	}

	buf := bytes.Buffer{}
	servers := strings.Split(strings.TrimSpace(upstreamStr), " ")
	for _, server := range servers {
		if server == "" {
			continue
		}
		buf.WriteString(fmt.Sprintf("nameserver %s\n", server))
	}

	return ssh.ActionList{
		ssh.DoMessageInfo("Using user-provided upstream DNS resolvers: %+v", servers),
		ssh.DoUploadBytesToFile(buf.Bytes(), common.DefResolvUpstreamConf),
	}
}
