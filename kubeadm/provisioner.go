package kubeadm

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		// ConnSchema: map[string]*schema.Schema{
		// 	"address": {
		// 		Type:     schema.TypeString,
		// 		Optional: true,
		// 	},
		// },

		Schema: map[string]*schema.Schema{
			"join": {
				Type:     schema.TypeString,
				Required: false,
			},
			"config": {
				Type:     schema.TypeString,
				Required: false,
			},
			"prevent_sudo": {
				Type:     schema.TypeBool,
				Required: false,
				Default:  false,
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
							Description: "user-provided installation script",
						},
						"version": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "kubeadm version to install.",
						},
					},
				},
			},
		},

		ApplyFunc: applyFn,
	}
}
