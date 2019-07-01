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
	"token": {
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
	////////////////////////////////////////////////////////////
	// certificates
	////////////////////////////////////////////////////////////
	"certs_secret": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "the key used for uploading the certificates to the cluster",
	},
	"certs_dir": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "the directory for certificates",
	},
	"certs_ca_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_ca_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_sa_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_sa_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_etcd_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_etcd_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_proxy_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"certs_proxy_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
}
