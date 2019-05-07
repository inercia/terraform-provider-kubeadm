## Introduction

Terraform cluster definition leveraging the libvirt provider.

## Pre-requisites

* _libvirt_

  The installation of libvirt is out of the
  scope of this document. Please refer
  to the instructions for your particular OS.

* _terraform/libvirt_ provider

  Follow the instuctions for installing
  the [Terraform/libvirt provider](https://github.com/dmacvicar/terraform-provider-libvirt)

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
