package main

import (
	"github.com/hashicorp/terraform/plugin"

	"github.com/inercia/terraform-provider-kubeadm/pkg/provisioner"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: provisioner.Provisioner,
	})
}
