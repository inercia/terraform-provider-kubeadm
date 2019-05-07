package provider

import (
	"fmt"
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

func dataSourceKubeadmRead(d *schema.ResourceData, meta interface{}) error {
	var err error

	token := d.Get("token").(string)
	if token == "" {
		log.Printf("[DEBUG] [KUBEADM] Generating a random token...")
		token, err = common.GetRandomToken()
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] [KUBEADM] kubeadm token = %s", token)
		d.Set("token", token)
	}

	if err := dataSourceKubeadmSetProvisionerConfig(d, token); err != nil {
		return err
	}

	d.SetId(token)

	return nil
}

// dataSourceKubeadmSetProvisionerConfig computes andd sets the config
// for the provisioner
func dataSourceKubeadmSetProvisionerConfig(d *schema.ResourceData, token string) error {
	var err error

	initConfig, err := dataSourceToInitConfig(d, token)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] [KUBEADM] init configuration:\n%s", initConfig)

	joinConfig, err := dataSourceToJoinConfig(d, token)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] [KUBEADM] join configuration:\n%s", joinConfig)

	if err := dataSourceVerify(d); err != nil {
		return err
	}

	// we must just copy some arguments from the provider configuration
	// to the provisioner configuration
	// FIXME: it seems we must convert everything to "strings", otherwise
	// Terraform just skips fields...
	provConfig := map[string]string{
		"init":                common.ToTerraformSafeString(initConfig[:]),
		"join":                common.ToTerraformSafeString(joinConfig[:]),
		"config_path":         d.Get("config_path").(string),
		"cni_plugin":          d.Get("cni.0.plugin").(string),
		"cni_plugin_manifest": d.Get("cni.0.plugin_manifest").(string),
		"helm_enabled":        fmt.Sprintf("%t", d.Get("addons.0.helm").(bool)),
		"dashboard_enabled":   fmt.Sprintf("%t", d.Get("addons.0.dashboard").(bool)),
	}
	d.Set("config", provConfig)

	log.Printf("[DEBUG] [KUBEADM] computing config for the provisioner:\n%s\n",
		spew.Sdump(d.Get("config")))

	// log.Printf("[DEBUG] [KUBEADM] provisioner configuration:\n%s\n", spew.Sdump(d))
	return nil
}

// dataSourceVerify verifies the config
func dataSourceVerify(d *schema.ResourceData) error {
	log.Printf("[DEBUG] [KUBEADM] veryfying configuration")

	hasKubeconfig := false
	if v, ok := d.GetOk("config_path"); ok && len(v.(string)) > 0 {
		hasKubeconfig = true
	}

	// check that, if we have a CNI plugin manifest, we have a valid kubeconfig
	if cniPluginManifestOpt, ok := d.GetOk("config.cni_plugin_manifest"); ok {
		cniPluginManifest := strings.ToLower(cniPluginManifestOpt.(string))
		if len(cniPluginManifest) > 0 && !hasKubeconfig {
			return fmt.Errorf("a CNI manifest is supposed to be loaded but the kubeconfig has not been specified with 'config_path'")
		}
	} else {
		if cniPluginOpt, ok := d.GetOk("cni.0.plugin"); ok {
			cniPlugin := strings.ToLower(cniPluginOpt.(string))
			if len(cniPlugin) > 0 {
				// check if a known CNI plugin has been provided
				log.Printf("[DEBUG] [KUBEADM] verifying CNI plugin %q is known", cniPlugin)
				if _, ok := common.CNIPluginsManifests[cniPlugin]; !ok {
					log.Printf("[DEBUG] [KUBEADM] CNI plugin: %s", cniPlugin)
					return fmt.Errorf("unknown CNI plugin %q", cniPlugin)
				}

				// check that we have a valid kubeconfig for loading the manifest
				if !hasKubeconfig {
					log.Printf("[DEBUG] [KUBEADM] CNI is supposed to be loaded but the kubeconfig has not been specified")
					return fmt.Errorf("CNI is supposed to be loaded but a local kubeconfig has not been specified with 'config_path'")
				}
			}
		}
	}

	return nil
}
