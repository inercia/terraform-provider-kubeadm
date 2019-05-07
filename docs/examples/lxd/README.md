## Introduction

Terraform cluster definition leveraging the libvirt provider.

## Pre-requisites

* _LXD_

  The easiest way to install LXD is with a Snap: https://snapcraft.io/lxd.
  Just do a `snap install lxd`. Then you will have to add your username to
  the `lxc` group (for accessing the LXD socket without being root).

* _terraform/LXD_

  You whill have to compile the LXD provider by yourself with a Golang compiler
  (and your `GOPATH` properly set).
  Do a `go get -v -u github.com/sl1pm4t/terraform-provider-lxd`.
  Maybe you will have to add the provider to your `~/.terraformrc` if terraform does not find
  the provider automatically. For example:
  ```
  providers {
    lxd = "/users/me/go/src/github.com/sl1pm4t/terraform-provider-lxd/terraform-provider-lxd"
  }
  ```

## Contents

* [Cluster definition](cluster.tf)
* [Variables](variables.tf)

## Machine access

By default all the machines will have the following users:

* All the instances have a `root` user with `linux` password.

## Topology

The cluster will be made by these machines:

  * X master nodes: have `kubeadm`, `kubelet` and `kubectl` preinstalled
  * Y worker nodes: have `kubeadm`, `kubelet` and `kubectl` preinstalled

All node should be able to ping each other and resolve their FQDN.
