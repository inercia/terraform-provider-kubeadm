# Terraform kubeadm plugin

[![Build Status](https://travis-ci.org/inercia/terraform-kubeadm.svg?branch=master)](https://travis-ci.org/inercia/terraform-provider-kubeadm)

This is a [Terraform](https://terraform.io/) _data_ definition and _provisioner_
that lets you install Kubernetes on a cluster. The underlying _resources_ could
be things like AWS instances, libvirt machines, LXD containers or any other
class of object that provides a SSH-like connection. The `kubeadm` `provisioner`
will run over the SSH connection all the commands necessary for installing
Kuberentes in those resources, according to the configuration specified in
the `data` block.

## Status

This provider/provisioner is still being actively developed. To see what is left
or planned, see the [issues list](https://github.com/inercia/terraform-provider-kubeadm/issues).

## Requirements

* Terraform

## Quick start

```console
$ mkdir -p $HOME/.terraform.d/plugins
$ go build -v -o $HOME/.terraform.d/plugins/terraform-provider-kubeadm \
    github.com/inercia/terraform-provider-kubeadm/cmd/terraform-provider-kubeadm
$ go build -v -o $HOME/.terraform.d/plugins/terraform-provisioner-kubeadm \
    github.com/inercia/terraform-provider-kubeadm/cmd/terraform-provisioner-kubeadm
```

## Usage

Here is an example that will setup Kubernetes in a cluster
created with the Terraform [libvirt](github.com/dmacvicar/terraform-provider-libvirt/)
provider:

```hcl
data "kubeadm" "main" {
  api {
    external = "loadbalancer.external.com"
  }
  
  cni {
    plugin = "flannel"
  }
  
  network {
    dns_domain = "my_cluster.local"  
    services = "10.25.0.0/16"
  }
  
  # install some extras: helm, the dashboard...
  addons {
    helm = "true"
    dashboard = "true"
  }
}

# from the libvirt provider
resource "libvirt_domain" "master" {
  name = "master"
  memory = 1024
  ...
  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"
    # there is no "join", so this will be the first node in the cluster
  }
}

# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"

    # this will make this minion _join_ the cluster started by the "master"
    join = "${libvirt_domain.master.network_interface.0.addresses.0}"
  }
}
```

Note well that:

* all the `provisioners` must specify the `config = ${XXX.config}`,
* any other nodes that _joins_ the _seeder_ must specify the
`join` attribute pointing to the `<IP/name>` they must _join_.

Now you can see the plan, apply it, and then destroy the
infrastructure:

```console
$ terraform plan
$ terraform apply
$ terraform destroy
```

You can find examples of the privider/provisioner in other environments like OpenStack, LXD, etc. in the [examples](docs/examples) directory)

## Documentation

* More details on the [installation](../../wiki/Installation) 
instructions.
* Using `kubeadm` in your Terraform scripts:
  * The [`data "kubeadm"`](../../wiki/Data_kubeadm) configuration
  block.
  * The [`provisioner "kubeadm"`](../../wiki/Provisioner_kubeadm)
  block.
  * [Additional stuff](../../wiki/Additional_tasks) ncessary for 
  having a fully functional Kubernetes cluster, like installing
  CNI, the dashboard, etc...
* [Roadmap, TODO and vision](../../wiki/Roadmap).

## Running acceptance tests

You need to define the TF_ACC variables:

```console
export TF_ACC=1
go test ./...
```

## Author(s)

* Alvaro Saurin \<alvaro.saurin@gmail.com\>

## License

* Apache 2.0, See LICENSE file
