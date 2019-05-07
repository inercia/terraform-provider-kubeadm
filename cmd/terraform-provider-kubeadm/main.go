package main

import (
	"github.com/hashicorp/terraform/plugin"

	"github.com/inercia/terraform-provider-kubeadm/pkg/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
