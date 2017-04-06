package main

import (
	"github.com/hashicorp/terraform/plugin"

	"github.com/inercia/terraform-kubeadm/provisioners/kubeadm"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: kubeadm.Provisioner,
	})
}
