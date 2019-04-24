# Terraform kubeadm plugin

[![Build Status](https://travis-ci.org/inercia/terraform-kubeadm.svg?branch=master)](https://travis-ci.org/inercia/terraform-provider-kubeadm)

This provider is still being actively developed. To see what is left or planned,
see the [issues list](https://github.com/inercia/terraform-provider-kubeadm/issues).

This is a terraform provider and provisioner that lets you install
kubernetes on a cluster provisioned with [Terraform](https://terraform.io/).

## Requirements

* Terraform

## Quick start

``` bash
$ go get -d github.com/inercia/terraform-provider-kubeadm
$ cd $GOPATH/src/github.com/inercia/terraform-provider-kubeadm
$ make
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
  
  network {
    dns_domain = "my_cluster.local"  
    services = "10.25.0.0/16"
  }
}

# from the libvirt provider
resource "libvirt_domain" "master" {
  name = "master"
  memory = 1024
  ...
  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config.init}"
  }
}

# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config.join}"
    join = "${libvirt_domain.master.network_interface.0.addresses.0}"
  }
}
```

Notice that the `provisioner` at the

* _seeder_ must specify the `config = ${XXX.config.init}`,
* any other nodes that _joins_ the _seeder_ must specify the
`config = ${XXX.config.join}` and a `join` pointing to the 
`<IP/name>` they must _join_.

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
