package common

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// ProvisionerConfigElements is the list of configuration options that can be
// passed from the provider to the provisioner
// FIXME: it seems we cannot use types other than "strings":
// Terraform just skips those fields otherwise
var ProvisionerConfigElements = map[string]*schema.Schema{
	"init": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"join": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"cni_plugin": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"cni_plugin_manifest": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"helm_enabled": {
		Type: schema.TypeBool,
		// Computed: true,
		Optional: true,
	},
	"dashboard_enabled": {
		Type: schema.TypeBool,
		// Computed: true,
		Optional: true,
	},
	"config_path": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
}
