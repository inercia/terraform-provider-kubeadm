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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"

	"github.com/inercia/terraform-provider-kubeadm/pkg/common"
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"config": {
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: common.ProvisionerConfigElements,
				},
			},
			"join": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "seeder node to join. Or start a seeder when not provided",
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				Description:  "role of this machine: master or worker",
				ValidateFunc: validation.StringInSlice([]string{"master", "worker"}, true),
			},
			"ignore_checks": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "list of preflight checks to ignore by kubeadm",
			},
			"drain": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "when true, remove this node from the cluster instead of adding it",
			},
			"nodename": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "name used for registering the node in the kubernetes cluster (defaults to the hostname)",
			},
			"listen": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "for masters, IP/DNS:port to listen at",
				ValidateFunc: common.ValidateHostPort,
			},
			"prevent_sudo": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "prevent the use of sudo",
			},
			"manifests": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "list of manifests to load in the API server once the master is setup",
			},
			"install": {
				// NOTE: default values for nested blocks are not available if the "install" block
				// has not been provided at all.
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"auto": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "try to automatically install kubeadm with the built-in helper script",
						},
						"script": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "script for installing kubeadm",
						},
						"inline": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "inline shell script code for installing kubeadm",
						},
						"version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "kubeadm version to install.",
						},
						"sysconfig_path": {
							Type:        schema.TypeString,
							Default:     common.DefKubeletSysconfigPath,
							Optional:    true,
							Description: fmt.Sprintf("full path for the uploaded kubelet sysconfig file (defaults to %s).", common.DefKubeletSysconfigPath),
						},
						"service_path": {
							Type:        schema.TypeString,
							Default:     common.DefKubeletServicePath,
							Optional:    true,
							Description: fmt.Sprintf("full path for the uploaded kubelet.service file (defaults to %s).", common.DefKubeletServicePath),
						},
						"dropin_path": {
							Type:        schema.TypeString,
							Default:     common.DefKubeadmDropinPath,
							Optional:    true,
							Description: fmt.Sprintf("full path for the uploaded kubeadm dropin file (defaults to %s).", common.DefKubeadmDropinPath),
						},
						"kubeadm_path": {
							Type:        schema.TypeString,
							Default:     common.DefKubeadmPath,
							Optional:    true,
							Description: "full path where kubeadm should be present (if no absolute path is provided, it will use the default PATH for finding it).",
						},
						"kubectl_path": {
							Type:        schema.TypeString,
							Default:     common.DefKubectlPath,
							Optional:    true,
							Description: "full path where kubectl should be present (if no absolute path is provided, it will use the default PATH for finding it).",
						},
					},
				},
			},
		},

		ApplyFunc: applyFn,

		// note: we cannot "validate" config passed from the provisioner, as the
		// validation is done before that config is created
	}
}

//
// Schema helpers
//

// getJoinFromResourceData returns the "joined host" from the ResourceData
func getJoinFromResourceData(d *schema.ResourceData) string {
	if opt, ok := d.GetOk("join"); ok {
		return strings.TrimSpace(opt.(string))
	}
	return ""
}

// getRoleFromResourceData returns the "role" host from the ResourceData
func getRoleFromResourceData(d *schema.ResourceData) string {
	if opt, ok := d.GetOk("role"); ok {
		return strings.TrimSpace(opt.(string))
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

func getSysconfigPathFromResourceData(d *schema.ResourceData) string {
	// NOTE: the "install" block is optional, so there will be no
	// default values for "install.0.XXX" if the "install" block has not been given...
	sysconfigPath := d.Get("install.0.sysconfig_path").(string)
	if len(sysconfigPath) == 0 {
		sysconfigPath = common.DefKubeletSysconfigPath
	}
	return sysconfigPath
}

func getServicePathFromResourceData(d *schema.ResourceData) string {
	servicePath := d.Get("install.0.service_path").(string)
	if len(servicePath) == 0 {
		servicePath = common.DefKubeletServicePath
	}
	return servicePath
}

func getDropinPathFromResourceData(d *schema.ResourceData) string {
	dropinPath := d.Get("install.0.dropin_path").(string)
	if len(dropinPath) == 0 {
		dropinPath = common.DefKubeadmDropinPath
	}
	return dropinPath
}

// getKubeadmFromResourceData returns the kubeadm binary path from the config
func getKubeadmFromResourceData(d *schema.ResourceData) string {
	if kubeadmPathOpt, ok := d.GetOk("install.0.kubeadm_path"); ok {
		return kubeadmPathOpt.(string)
	}
	return common.DefKubeadmPath
}

// getTokenFromResourceData returns the current token in the ResourceData
func getTokenFromResourceData(d *schema.ResourceData) string {
	if configOpt, ok := d.GetOk("config"); ok {
		config := configOpt.(map[string]interface{})
		if t, ok := config["token"]; ok {
			return t.(string)
		}
	}
	return ""
}

// getKubectlFromResourceData returns the kubectl binary path from the config
func getKubectlFromResourceData(d *schema.ResourceData) string {
	if kubectlPathOpt, ok := d.GetOk("install.0.kubectl_path"); ok {
		return kubectlPathOpt.(string)
	}
	return common.DefKubectlPath
}

// getNodenameFromResourceData returns the nodename specified in the ResourceData
func getNodenameFromResourceData(d *schema.ResourceData) string {
	if nodenameOpt, ok := d.GetOk("nodename"); ok {
		return nodenameOpt.(string)
	}
	return ""
}
