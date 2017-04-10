# Terraform kubeadm plugin

[![Build Status](https://travis-ci.org/inercia/terraform-kubeadm.svg?branch=master)](https://travis-ci.org/inercia/terraform-kubeadm)

This provider is still being actively developed. To see what is left or planned,
see the [issues list](https://github.com/inercia/terraform-kubeadm/issues).

This is a terraform provider and provisioner that lets you install
kubernetes on a cluster provisioned with [Terraform](https://terraform.io/).

## Requirements

* Terraform

## Installing

### ... from RPMs

[Copied from the Terraform documentation](https://www.terraform.io/docs/plugins/basics.html):
> To install a plugin, put the binary somewhere on your filesystem, then configure Terraform to be able to find it. The configuration where plugins are defined is ~/.terraformrc for Unix-like systems and %APPDATA%/terraform.rc for Windows.

If you are using opensuse/SUSE distro, add the repo and download the package (check the repo according your distro)

```console

DISTRO=openSUSE_Leap_42.1
zypper addrepo http://download.opensuse.org/repositories/Virtualization:containers/$DISTRO/Virtualization:containers.repo
zypper refresh
zypper install terraform-kubeadm

```

### ... from source

1.  `go get -u github.com/inercia/terraform-kubeadm`

2.  Switch the terraform project back to the stable version, otherwise you will get a `Incompatible API version with plugin. Plugin version: 1, Ours: 2` error at runtime:
    ```
    cd $GOPATH/src/github.com/hashicorp/terraform
    git checkout v0.6.16
    cd $GOPATH/src/github.com/inercia/terraform-kubeadm
    ```
3.  .. or alternatively install [govend](https://github.com/govend/govend) and:
    1. Run `govend`, which will scan dependencies and download them into vendor
    2. problematic dependencies, like terraform, will be automatically in the right version thanks to the `vendor.yml` file.
4.  Run `go install` to build the binary. You will now find the
    binary at `$GOPATH/bin/terraform-provider-libvirt`.

### `terraformrc` file

Even though Terraform has an autodiscovery mechanism for finding plugins, you should _register_ this plugins
by adding it to your `~/.terraformrc` file, keeping any previous plugins you could have. For example,
your `~/.terraformrc` could look like this:

```hcl
providers {
  libvirt = "/home/user/go/bin/terraform-provider-libvirt"
  kubeadm = "/home/user/go/bin/terraform-kubeadm"
}

provisioners {
  kubeadm = "/home/user/go/bin/terraform-kubeadm"
}
```

## Usage

Here is an example that will setup Kubernetes in a cluster
created with the Terraform [libvirt](github.com/dmacvicar/terraform-provider-libvirt/)
provider:

```hcl
resource "kubeadm" "main" {
  dns_domain = "my_cluster"
  services_cidr = "10.25.0.0/16"
}

# from the libvirt provider
resource "libvirt_domain" "master" {
  name = "master"
  memory = 1024
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.k8s.config.master}"
  }
}

# from the libvirt provider
resource "libvirt_domain" "minion" {
  count      = 3
  name       = "minion${count.index}"
  ...
  provisioner "kubeadm" {
    config = "${kubeadm.k8s.config.node}"
    master = "${libvirt_domain.master.network_interface.0.addresses.0}"
  }
}
```

Notice that the `provisioner` at the

* _master_ must specify the `config = ${... config.master}`,
* _nodes_ must specify the `config = ${... config.node}` and a `master` pointing
to the `<IP/name>` of the _master_

Now you can see the plan, apply it, and then destroy the infrastructure:

```console
$ terraform plan
$ terraform apply
$ terraform destroy
```

## Arguments

This is the list of arguments you can use in the `resource "kubeadm"`:

  * `api_advertised`: API server advertised IP/name
  * `api_port`: API server binding port
  * `api_alt_names`: List of SANs to use in api-server certificate. Example: 'IP=127.0.0.1,IP=127.0.0.2,DNS=localhost', If empty, SANs will be extracted from the api_servers
  * `authorization_mode`: Authentication mode (Example: `RBAC`)
  * `cloud_provider`: The provider for cloud services.  Empty string for no provider
  * `etcd_servers`: List of etcd servers URLs including host:port
  * `dns_domain`: The DNS domain
  * `pods_cidr`: The CIDR range of cluster pods
  * `services_cidr`: The CIDR range of cluster services (Example: `10.3.0.0/24`)
  * `version`: Kubernetes version to use (Example: `v1.6.1`)

## Known limitations

* `kubeadm 1.6.[01]` seems to be broken for me, so you should use
images (or repos) with kubernetes `1.5.x`. But running kubeadm `1.5` does
not mean you cannort run kubernetes 1.6: just use `version = 1.6` as a
cluster argument.
* The `kubeadm-setup.sh` tries to does its best in order to install
`kubeadm`, but some distros have not been tested too much. I'use used
`libvirt` with _OpenSUSE Leap 42.2_ images for running my tests, so that
could be considered the perfect combination for trying this...

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
