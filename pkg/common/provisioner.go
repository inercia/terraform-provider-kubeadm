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

package common

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Rationale:
//
// The "provisioner" does not have access to the "resource kubeadm", so we
// must pass configuration in some way from one to the other.
//
// ProvisionerConfigElements is the list of configuration options that are
// passed from the "provider" to the "provisioner".
//
// This dictionary is passed to templates as well, so you can use ie, {{.token}}
// or {{.flannel_backend}} in the templates that loaded in the cluster later on
// (for exmaple, in the CNI manifest)
//
// FIXME: it seems we cannot use types other than "strings": Terraform just skips those fields otherwise
//
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
		Optional:  true,
		Sensitive: true,
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
	"cni_conf_dir": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"cni_pod_cidr": {
		Type: schema.TypeString,
		// Computed: true,
		Optional: true,
	},
	"flannel_backend": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "the flannel backend",
	},
	"flannel_image_version": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "the flannel image version",
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
	"certs_dir": {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "the directory for certificates",
	},
	////////////////////////////////////////////////////////////
	// certificates
	////////////////////////////////////////////////////////////
	"ca_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"ca_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"sa_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"sa_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"etcd_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"etcd_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"proxy_crt": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
	"proxy_key": {
		Type: schema.TypeString,
		// Computed: true,
		Optional:  true,
		Sensitive: true,
	},
}
