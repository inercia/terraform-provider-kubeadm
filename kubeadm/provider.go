package kubeadm

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"kubeadm": dataSourceKubeadm(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"kubeadm": schema.DataSourceResourceShim(
				"kubeadm",
				dataSourceKubeadm(),
			),
		},
	}
}
