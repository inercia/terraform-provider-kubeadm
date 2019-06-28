package provider

import (
	"fmt"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

// dataSourceKubeadmCreate is responsible for creating the kubeadm configuration and certificates
func dataSourceKubeadmCreate(d *schema.ResourceData, meta interface{}) error {
	// NOTE: this function is called even for doing a "terraform plan"

	log.Printf("[DEBUG] [KUBEADM] dataSourceKubeadmRead: new resource = %v", d.IsNewResource())

	token := d.Get("token").(string)
	// FIXME: users can provide their own token: we should detect we need to initialize all the other things
	if token == "" {
		log.Printf("[DEBUG] [KUBEADM] no previous configuration found: creating new configuration...")
		if err := createConfigForProvisioner(d); err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] [KUBEADM] using previous kubeadm token = %s", token)
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
		log.Printf("[DEBUG] [KUBEADM] trying to remove current kubeconfig file %q", kubeconfigS)
		err := os.Remove(kubeconfigS)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// dataSourceKubeadmExists checks if the kubeadm configuration already exists
func dataSourceKubeadmExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Printf("[DEBUG] [KUBEADM] checking if kubeadm configuration already exists...")

	// check we have the token
	token, ok := d.GetOk("token")
	if !ok {
		log.Printf("[DEBUG] [KUBEADM] does not exist: no token")
		return false, nil
	}

	if token.(string) == "" {
		log.Printf("[DEBUG] [KUBEADM] does not exist: no token")
		return false, nil
	}

	// check we have the certificates
	_, ok = d.GetOk("config")
	if !ok {
		log.Printf("[DEBUG] [KUBEADM] does not exist: no config section")
		return false, nil
	}

	certsConfig := common.CertsConfig{}
	err := certsConfig.FromResourceData(d)
	if err != nil {
		log.Printf("[DEBUG] [KUBEADM] does not exist: no certs config")
		return false, err
	}

	if !certsConfig.IsFilled() {
		log.Printf("[DEBUG] [KUBEADM] does not exist: empty certs")
		return false, nil
	}

	return true, nil
}

// createConfigForProvisioner computes and sets the config for the provisioner
func createConfigForProvisioner(d *schema.ResourceData) error {
	var err error

	log.Printf("[DEBUG] [KUBEADM] generating a random token...")
	token, err := common.GetRandomToken()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] [KUBEADM] setting %s as the ID for the kubeadm 'data'", token)
	d.SetId(token)

	log.Printf("[DEBUG] [KUBEADM] kubeadm token = %s", token)
	if err = d.Set("token", token); err != nil {
		return err
	}

	log.Printf("[DEBUG] [KUBEADM] creating kubeadm configuration")

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
	log.Printf("[DEBUG] [KUBEADM] init configuration:")
	log.Printf("[DEBUG] [KUBEADM] ------------------------")
	log.Printf("[DEBUG] [KUBEADM] \n%s", string(initConfigBytes))
	log.Printf("[DEBUG] [KUBEADM] ------------------------")

	joinConfigBytes, err := common.JoinConfigToYAML(joinConfig)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] [KUBEADM] join configuration:")
	log.Printf("[DEBUG] [KUBEADM] ------------------------")
	log.Printf("[DEBUG] [KUBEADM] \n%s", string(joinConfigBytes))
	log.Printf("[DEBUG] [KUBEADM] ------------------------")

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
		"helm_enabled":        fmt.Sprintf("%t", d.Get("addons.0.helm").(bool)),
		"dashboard_enabled":   fmt.Sprintf("%t", d.Get("addons.0.dashboard").(bool)),
	}

	// create all the certs and set them in the `d.config`, so the provisioner
	// will configure kubeadm for uploading them to the API server once the cluster
	// is running. We will also set the encryption key here.
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

	log.Printf("[DEBUG] [KUBEADM] -------------------------------------------------------------------------")
	log.Printf("[DEBUG] [KUBEADM] 'data.config' after configuration:")
	log.Printf("[DEBUG] [KUBEADM] %s", spew.Sdump(provConfig))
	log.Printf("[DEBUG] [KUBEADM] end of provisioner config.")
	log.Printf("[DEBUG] [KUBEADM] -------------------------------------------------------------------------")

	return nil
}

// dataSourceVerify verifies the config
func dataSourceVerify(d *schema.ResourceData) error {
	log.Printf("[DEBUG] [KUBEADM] verifying configuration...")
	// Nothing to do at this time...
	log.Printf("[DEBUG] [KUBEADM] ... configuration seems to be fine.")
	return nil
}
