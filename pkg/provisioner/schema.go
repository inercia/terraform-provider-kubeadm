package provisioner

import (
	"fmt"

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
			// not sure really necessary: maybe we can get Changes('count'):
			"remove": {
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
							Description: "full path where kubeadm should be present (if no absolute path is provided, it will use the defaultt PATH for finding it).",
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
