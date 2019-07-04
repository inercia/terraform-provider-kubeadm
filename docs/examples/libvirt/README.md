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

* `kubectl`

  A local kubectl executable.

## Contents

* [Cluster definition](cluster.tf)
* [Variables](variables.tf)

## Machine access

By default all the machines will have the following users:

* All the instances have a `root` user with `linux` password.

## Topology

The topology created for libvirt is currently a bit limited:

  * only one master, with `kubeadm` and the `kubelet` pre-installed.
  No load balancer is created, so you are limited to only one master.
  * `${var.worker_count}` worker nodes, with `kubeadm` and the `kubelet` pre-installed.


You should be able to `ssh` these machines, and all of them should be able to ping each other.
