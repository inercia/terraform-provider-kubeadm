# Terraform kubeadm plugin

[![Build Status](https://travis-ci.org/inercia/terraform-provider-kubeadm.svg?branch=master)](https://travis-ci.org/inercia/terraform-provider-kubeadm)

A [Terraform](https://terraform.io/) `resource` definition and `provisioner`
that lets you install Kubernetes on a cluster.

The underlying `resources` where the `provisioner` runs could be things like
AWS instances, `libvirt` machines, LXD containers or any other
resource that supports SSH-like connections. The `kubeadm` `provisioner`
will run over this SSH connection all the commands necessary for installing
Kubernetes in those resources, according to the configuration specified in
the `resource "kubeadm"` block.

## Example

Here is an example that will setup Kubernetes in a cluster
created with the Terraform [libvirt](github.com/dmacvicar/terraform-provider-libvirt/)
provider:

```hcl
resource "kubeadm" "main" {
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
  
  # this provisioner will start a Kubernetes master in this machine,
  # with the help of "kubeadm" 
  provisioner "kubeadm" {
    # there is no "join", so this will be the first node in the cluster: the seeder
    config = "${kubeadm.main.config}"

    # when creating multiple masters, the first one (the _seeder_) must join="",
    # and the rest will join it afterwards...
    join      = "${count.index == 0 ? "" : libvirt_domain.master.network_interface.0.addresses.0}"
    role      = "master"

    install {
      # this will try to install "kubeadm" automatically in this machine
      auto = true
    }
  }

  # provisioner for removing the node from the cluster
  provisioner "kubeadm" {
    when   = "destroy"
    config = "${kubeadm.main.config}"
    drain  = true
  }
}

# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  
  # this provisioner will start a Kubernetes worker in this machine,
  # with the help of "kubeadm"
  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"

    # this will make this minion "join" the cluster started by the "master"
    join = "${libvirt_domain.master.network_interface.0.addresses.0}"
    install {
      # this will try to install "kubeadm" automatically in this machine
      auto = true
    }
  }

  # provisioner for removing the node from the cluster
  provisioner "kubeadm" {
    when   = "destroy"
    config = "${kubeadm.main.config}"
    drain  = true
  }
}
```

Note well that:

* all the `provisioners` must specify the `config = ${kubeadm.XXX.config}`,
* any other nodes that _join_ the _seeder_ must specify the
`join` attribute pointing to the `<IP/name>` they must _join_. You can use
the optional `role` parameter for specifying whether it is joining as a
`master` or as a `worker`. 

Now you can see the plan, apply it, and then destroy the
infrastructure:

```console
$ terraform plan
$ terraform apply
$ terraform destroy
```

You can find examples of the privider/provisioner in other environments like OpenStack, LXD, etc. in the [examples](docs/examples) directory)

## Features

* Easy deployment of kubernetes clusters in any platform supported
by Terraform, just adding our `provisioner "kubeadm"` in the machines
you want to be part of the cluster.
  * All operations are performed through the SSH connection created by Terraform, 
  so you can create a k8s cluster in completely isolated machines.
* Multi-master deployments. Just add a Load Balancer that points
to your masters and you will have a HA cluster!.  
* Easy _scale-up_/_scale-down_ of the cluster by just changing the
`count` of your masters or workers.
* Use the [`kubeadm` attributes](../../wiki/Resource_kubeadm#attributes-reference)
in other parts of your Terraform script. This makes it easy to do things like:
  * enabling SSL termination by using the certificates generated for `kubeadm`
   in the code you have for creating your Load Balancer.
  * create machine _templates_ (for example, `cloud-init` code) that can 
  be used for creating machines dynamically, without Terraform being involved
  (like _autoscaling groups_ in AWS).
* Automatic rolling upgrade of the cluster by just changing the base
image of your machines. Terraform will take care of replacing old
nodes with upgraded ones, and this provider will take care of draining
the nodes.
* Automatic deployment of some _addons_, like CNI drivers, the k8s Dashboard,
Helm, etc.  

(check the [TODO](../../wiki/Roadmap) for an updated list of features).  

## Status

This `provider`/`provisioner` is being actively developed, but I would still consider
it **ALPHA**, so there can be many rough edges and some things can change without
any previous notice. To see what is left or planned, see the
[issues list](https://github.com/inercia/terraform-provider-kubeadm/issues) and the
[roadmap](../../wiki/Roadmap).

## Requirements

* [Terraform](https://www.terraform.io)
* Go >= 1.12 (for compiling)

## Quick start

```console
$ mkdir -p $HOME/.terraform.d/plugins
$ # with go>=1.12
$ go build -v -o $HOME/.terraform.d/plugins/terraform-provider-kubeadm \
    github.com/inercia/terraform-provider-kubeadm/cmd/terraform-provider-kubeadm
$ go build -v -o $HOME/.terraform.d/plugins/terraform-provisioner-kubeadm \
    github.com/inercia/terraform-provider-kubeadm/cmd/terraform-provisioner-kubeadm
```

## Documentation

* More details on the [installation](../../wiki/Installation) 
instructions.
* Using `kubeadm` in your Terraform scripts:
  * The [`resource "kubeadm"`](../../wiki/Resource_kubeadm) configuration
  block.
  * The [`provisioner "kubeadm"`](../../wiki/Provisioner_kubeadm)
  block.
  * [Additional stuff](../../wiki/Additional_tasks) ncessary for 
  having a fully functional Kubernetes cluster, like installing
  CNI, the dashboard, etc...
* Deployment examples for:
  * [AWS](docs/examples/aws/README.md)
  * [libvirt](docs/examples/libvirt/README.md)
  * [lxd](docs/examples/lxd/README.md)
  * [Docker-in-Docker](docs/examples/dnd/README.md)
* [Roadmap, TODO and vision](../../wiki/Roadmap).
* [FAQ](../../wiki/FAQ).

## Running the tests

You can run the unit tests with:

```console
$ make test
```

There are [end-to-end tests](tests/e2e) as well, that can be launched with

```console
$ make tests-e2e
```

## Author(s)

* Alvaro Saurin \<alvaro.saurin@gmail.com\>

## License

* Apache 2.0, See LICENSE file
