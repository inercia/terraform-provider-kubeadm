# Terraform kubeadm plugin

[![Build Status](https://travis-ci.org/inercia/terraform-kubeadm.svg?branch=master)](https://travis-ci.org/inercia/terraform-kubeadm)

This provider is still being actively developed. To see what is left or planned,
see the [issues list](https://github.com/inercia/terraform-kubeadm/issues).

This is a terraform provider and provisioner that lets you install
kubernetes on a cluster provisioned with [Terraform](https://terraform.io/).

## Requirements

* Terraform

## Installing

### ... from source

1.  `go get -u github.com/inercia/terraform-kubeadm`

2.  Make sure your Terraform binary has been built with some stable version,
    otherwise you will get a `Incompatible API version with plugin. Plugin version: 1, Ours: 2` error at runtime. If you built it from sources:
    ```
    cd $GOPATH/src/github.com/hashicorp/terraform
    git checkout v0.8.0
    cd $GOPATH/src/github.com/inercia/terraform-kubeadm
    ```
3.  Run `make` to build the binaries. You will now find the
    binary at `$GOPATH/bin/terraform-{provider,provisioner}-kubeadm`.

### `terraformrc` file

Even though Terraform has an autodiscovery mechanism for finding plugins, you should _register_ this plugins
by adding it to your `~/.terraformrc` file, keeping any previous plugins you could have. For example,
your `~/.terraformrc` could look like this:

```hcl
providers {
  libvirt = "/home/user/go/bin/terraform-provider-libvirt"
  kubeadm = "/home/user/go/bin/terraform-provider-kubeadm"
}

provisioners {
  kubeadm = "/home/user/go/bin/terraform-provisioner-kubeadm"
}
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

* _seeder_ must specify the `config = ${XXX.config}`,
* _joiner_ must specify the `config = ${XXX.config}` and a `join` pointing
to the `<IP/name>` they must _join_.

Now you can see the plan, apply it, and then destroy the infrastructure:

```console
$ terraform plan
$ terraform apply
$ terraform destroy
```

## Arguments

### ... for the provisioner

  * `join`: the address of the node to join in the cluster. 
  The absence of a `join` indicates that this node will be used as a kubernetes
  master and seeder for the cluster.
  * `config`: a reference to the `config` configuration of the _provider_.
  * `install`: (true/false) try to install the `kubeadm` package with the help of
  the built-in script.
  * `install_version` (optional): the version of kubeadm installed by automatic
  `kubeadm` installer.
  * `install_script` (optional): a user-provider script that will be used for installing
  `kubeadm`. It will be uploaded to all the machines in the cluster and executed
  before trying to run `kubeadm`.
  It can be `v1.5` (default) or `v1.6`.

## Known limitations

* There is currently no way for downloading the `kubeconfig` file generated
by `kubeadm`. You must `ssh` to the master machine and get the file from
`/etc/kubernetes/admin.conf`.
* `kubeadm` currently does not install any networking driver (ie, `flannel`,
`calico`, etc). You need a valid `kubeconfig`, and then you can install the
driver by just invoking `kubectl` with the right manifest, for example (for
Flannel):
  ```
  export ARCH=amd64
  export KUBECONFIG=<your-kubeconfig-file>
  curl -sSL "https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml?raw=true" | sed "s/amd64/${ARCH}/g" | kubectl create -f -
  ```
* The `kubeadm-setup.sh` tries to does its best in order to install
`kubeadm`, but some distros have not been tested too much. I'use used
`libvirt` with _OpenSUSE Leap 42.2_ images for running my tests, so that
could be considered the perfect combination for trying this...
* See also the current list of [`kubeadm` limitations](https://kubernetes.io/docs/getting-started-guides/kubeadm/#limitations)

## Running acceptance tests

You need to define the TF_ACC variables:

```console
export TF_ACC=1
go test ./...
```

## Author(s)

* Alvaro Saurin <alvaro.saurin@suse.de>

## License

* Apache 2.0, See LICENSE file
