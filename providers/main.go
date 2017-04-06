package main

import (
	"github.com/hashicorp/terraform/plugin"

	"github.com/inercia/terraform-kubeadm/providers/kubeadm"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc:    kubeadm.Provider,
	})
}
