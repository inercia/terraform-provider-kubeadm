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

package provider

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/internal/ssh"
	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// dataSourceKubeadmCreate is responsible for creating the kubeadm configuration and certificates
func dataSourceKubeadmCreate(d *schema.ResourceData, meta interface{}) error {
	ssh.Debug("dataSourceKubeadmRead: new resource = %v", d.IsNewResource())

	_, ok := d.GetOk("config")
	if !ok {
		ssh.Debug("no previous configuration found: creating new configuration...")
		if err := createConfigForProvisioner(d); err != nil {
			return err
		}
	} else {
		ssh.Debug("using previous config")
	}

	if err := dataSourceVerify(d); err != nil {
		return err
	}

	return dataSourceKubeadmRead(d, meta)
}

// dataSourceKubeadmReads is responsible for reading any resources
func dataSourceKubeadmRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

// dataSourceKubeadmDelete is responsible for deleting all the kubeadm resources
func dataSourceKubeadmDelete(d *schema.ResourceData, meta interface{}) error {
	kubeconfig, ok := d.GetOk("config_path")
	if ok {
		kubeconfigS := kubeconfig.(string)
		ssh.Debug("trying to remove current kubeconfig file %q", kubeconfigS)
		err := os.Remove(kubeconfigS)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// dataSourceKubeadmUpdate is responsible for updating things
func dataSourceKubeadmUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO: pass the responsability for creating the new token to the provisioner
	return nil
}

// dataSourceKubeadmExists checks if the kubeadm configuration already exists
func dataSourceKubeadmExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ssh.Debug("checking if kubeadm configuration already exists...")

	// check we have the certificates
	_, ok := d.GetOk("config")
	if !ok {
		ssh.Debug("does not exist: no config section")
		return false, nil
	}

	certsConfig := common.CertsConfig{}
	err := certsConfig.FromResourceDataConfig(d)
	if err != nil {
		ssh.Debug("does not exist: no certs config")
		return false, err
	}

	if !certsConfig.HasAllCertificates() {
		ssh.Debug("does not exist: empty certs")
		return false, nil
	}

	return true, nil
}

// createConfigForProvisioner computes and sets the config for the provisioner
func createConfigForProvisioner(d *schema.ResourceData) error {
	var err error

	ssh.Debug("generating a random token...")
	token, err := common.GetRandomToken()
	if err != nil {
		return err
	}
	ssh.Debug("kubeadm token = %s", token)

	ssh.Debug("creating kubeadm configuration for init and join")
	initConfig, err := dataSourceToInitConfig(d, token)
	if err != nil {
		return err
	}
	joinConfig, err := dataSourceToJoinConfig(d, token)
	if err != nil {
		return err
	}

	initConfigBytes, err := common.InitConfigToYAML(initConfig)
	if err != nil {
		return err
	}
	ssh.Debug("init configuration:")
	ssh.Debug("------------------------")
	ssh.Debug("\n%s", string(initConfigBytes))
	ssh.Debug("------------------------")

	joinConfigBytes, err := common.JoinConfigToYAML(joinConfig)
	if err != nil {
		return err
	}
	ssh.Debug("join configuration:")
	ssh.Debug("------------------------")
	ssh.Debug("\n%s", string(joinConfigBytes))
	ssh.Debug("------------------------")

	kubeconfig := d.Get("config_path").(string)

	// we must just copy some arguments from the provider configuration
	// to the provisioner configuration
	// FIXME: it seems we must convert everything to "strings", otherwise
	// Terraform just skips fields...
	// NOTE: these fields must be in ProvisionerConfigElements
	provConfig := map[string]interface{}{
		"token":               token,
		"init":                common.ToTerraformSafeString(initConfigBytes[:]),
		"join":                common.ToTerraformSafeString(joinConfigBytes[:]),
		"config_path":         kubeconfig,
		"cni_plugin":          d.Get("cni.0.plugin").(string),
		"cni_plugin_manifest": d.Get("cni.0.plugin_manifest").(string),
		"helm_enabled":        fmt.Sprintf("%t", d.Get("helm.0.install").(bool)),
		"dashboard_enabled":   fmt.Sprintf("%t", d.Get("dashboard.0.install").(bool)),
		"certs_dir":           initConfig.CertificatesDir,
	}

	if cniConfigDir, ok := d.GetOk("cni.0.conf_dir"); ok {
		provConfig["cni_conf_dir"] = cniConfigDir.(string)
	} else {
		provConfig["cni_conf_dir"] = common.DefCniConfDir
	}

	if cniBinDir, ok := d.GetOk("cni.0.bin_dir"); ok {
		provConfig["cni_bin_dir"] = cniBinDir.(string)
	} else {
		provConfig["cni_bin_dir"] = common.DefCniBinDir
	}

	if p, ok := d.GetOk("network.0.pods"); ok {
		provConfig["cni_pod_cidr"] = p.(string)
	} else {
		provConfig["cni_pod_cidr"] = common.DefPodCIDR
	}

	if fb, ok := d.GetOk("cni.0.flannel.0.backend"); ok {
		provConfig["flannel_backend"] = fb.(string)
	} else {
		provConfig["flannel_backend"] = common.DefFlannelBackend
	}

	if v, ok := d.GetOk("cni.0.flannel.0.version"); ok {
		provConfig["flannel_image_version"] = v.(string)
	} else {
		provConfig["flannel_image_version"] = common.DefFlannelImageVersion
	}

	if version, ok := d.GetOk("version"); ok {
		provConfig["kube_version"] = version.(string)
	} else {
		provConfig["kube_version"] = common.DefKubernetesVersion
	}

	if cloudProviderRaw, ok := d.GetOk("cloud.0.provider"); ok && len(cloudProviderRaw.(string)) > 0 {
		cloudProvider := cloudProviderRaw.(string)
		provConfig["cloud_provider"] = cloudProvider

		// check if have some extra flags...
		if managerFlagsRaw, ok := d.GetOk("cloud.0.manager_flags"); ok && len(managerFlagsRaw.(string)) > 0 {
			managerFlags := managerFlagsRaw.(string)
			provConfig["cloud_provider_flags"] = managerFlags
		}

		// ... and maybe if we have some cloud-provider config file
		if cloudConfigRaw, ok := d.GetOk("cloud.0.config"); ok && len(cloudConfigRaw.(string)) > 0 {
			cloudConfig := cloudConfigRaw.(string)
			provConfig["cloud_config"] = common.ToTerraformSafeString([]byte(cloudConfig))
		}
	}

	// create all the certs and set them in some `d.config` fields, so the provisioner
	// can upload them to the machines in the Control Plane
	certConfig, err := common.CreateCerts(d, initConfig)
	if err != nil {
		return err
	}
	for k, v := range certConfig {
		provConfig[k] = v
	}

	if err = d.Set("config", provConfig); err != nil {
		return err
	}

	ssh.Debug("-------------------------------------------------------------------------")
	ssh.Debug("'data.config' after configuration:")
	ssh.Debug("%s", spew.Sdump(provConfig))
	ssh.Debug("end of provisioner config.")
	ssh.Debug("-------------------------------------------------------------------------")

	// create the ID as the hash of the init and join configurations
	hasher := md5.New()
	hasher.Write(initConfigBytes[:])
	hasher.Write(joinConfigBytes[:])
	d.SetId(hex.EncodeToString(hasher.Sum(nil)))

	return nil
}

// dataSourceVerify verifies the config
func dataSourceVerify(d *schema.ResourceData) error {
	ssh.Debug("verifying configuration...")
	// Nothing to do at this time...
	ssh.Debug("... configuration seems to be fine.")
	return nil
}
