## Introduction

Terraform cluster definition leveraging the Docker provider, using Docker
containers as Kubernetes nodes.

This configuration can be used for having a convenient way to start a
Kubernetes cluster in your local laptop, using regular Docker containers
as nodes of your cluster.

![Run](run-example.svg)

## How does it work?

The Docker daemon can be run _in_ a Docker container in what is usually called
a _DnD_ (_Docker-in-Docker_) configuration. This requires a special [Dockerfile](Dockerfile)
that has been tweaked for starting `systemd` as the the _entrypoint_. `systemd` will them start
the Docker daemon as well as the `kubelet`. Once all these elements are running, we can
run `kubeadm` as in any other platform for starting a Kubernetes cluster. 

## Pre-requisites

* _Docker_

  You will need a functional Docker daemon running. Make sure the `${var.daemon}`
  is properly set, pointing to a daemon where you can launch containers. 

* `kubectl`

  A local kubectl executable.

## Contents

* [Cluster definition](cluster.tf)
* [Variables](variables.tf)

## Machine access

By default all the machines will have the following users:

* All the instances have a `root` user with `linux` password.

## Topology

The cluster will be made by these machines:

  * `${var.master_count}` master nodes, with `kubeadm` and the `kubelet` pre-installed.
  * `${var.worker_count}` worker nodes, with `kubeadm` and the `kubelet` pre-installed.

You should be able to `ssh` these machines, and all of them should be able to ping each other.

## Status

There is a bug in the Terraform Docker provider that
prevents containers from being stopped when using `rm=true`,
so there are some problems when re-creating resources.
