## Introduction

Example code for creating a Kubernetes cluster in AWS with the
help of the kubeadm provider.

## Pre-requisites

* Some valid credentials (key and secret) for accessing AWS.
 
## Contents

* [Cluster definition](cluster.tf)
* [Variables](variables.tf)

## Machine access

Depending on the distro used (see the variable `var.ami_distro`),
instances have the default user. For Ubuntu it will be the `ubuntu`
user, while `fedora` for the Fedora distro.

All nodes should be accessible by jumping through a _bastion host_
with something like:

```bash
$ ssh -J ubuntu@<BASTION-IP> ubuntu@<NODE-IP>
```

where these IP addresses can be obtained after `apply`ing
with `terraform output`. 

The private/public keys used for accessing all the instances (as
well as the bastion host) can be customizable with
`var.private_key`, being the default value 
`~/.ssh/id_rsa`/`~/.ssh/id_rsa.pub`.

## Usage

Example:

```bash
$ TF_VAR_stack_name="alv-k8s" \
  TF_VAR_aws_access_key="AKIAZ..." \
  TF_VAR_aws_secret_key="sto3ybm+j..." \
  TF_VAR_private_key=~/.ssh/aws \
  terraform apply -auto-approve
```

You could be intersested in customizing some variables like:

* `stack_name`: identifier to make all your resources unique and avoid
clashes with other users of this Terraform project.
* `private_key`: the filename of a `ssh` private key used for accessing
all the nodes (a corresponding `.pub` file must exist too).
* `ami_distro`: the Linux distro to use, currently `ubuntu` or `fedora`.
* `aws_region`: name of the region to be used.
* `aws_az`: the AWS Availability Zone
* `vpc_cidr`, `vpc_cidr_public`, `vpc_cidr_private`: the subnet CIDRs
for the VPC and the public and private subnets.
* `master_size` and `worker_size`: the VM size for the masters
and workers.
 
## Topology

The cluster will be made by these machines:

  * a bastion host, with a public IP, used for accesing
  the nodes in the cluster.
  * `var.masters` master nodes (by default, one) 
  * `var.workers` worker nodes (by default, one)

## Some notes on the Terraform code

There are some constraints imposed by Terraform/AWS that must be
taken into account when using this code as a base for your own
deployments. 

1) Firstly, the cluster creation order in AWS is not _natural_
   for using our provisioner. The problem comes when `kubeadm` is started
   in the seeder as part of the provisioning. At some point it tries to
   access the API server through the Load Balancer. However, the Load
   Balancer creation depends on the very same instance being fully
   created _and provisioned_, so there is a deadlock in the
   creation process.

   The only solution is to do the `kubeadm` provisioning _after_
   the instance and the Load Balancer have been created, using a
   `null_resource` where the `provisioner "kubeadm"` is embeded.

2) In addition, the Load Balancer checks that the API server(s) are healthy 
   for keeping them in the list of backends. So if we create `1)` a master
   instance `2)` the Load Balancer `3)` the API server in that master, there is
   a lapse of time when the API server is not going to respond to the health
   checks. By using an higher `unhealthy_threshold` in the `health_check`
   we can reduce this chance of failure, but we are also reducing
   the ability to detect real failures in the backend(s).
 
3) Nodes must be registered with the private DNS names provided
   by AWS.