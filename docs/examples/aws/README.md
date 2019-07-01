## Introduction

Example code for creating a Kubernetes cluster in AWS with the
help of the kubeadm provider.

## Pre-requisites

 * Some valid credentials (key and secret) for accessing AWS.
 
 * `kubectl`
  
   A local kubectl executable.

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

Deployment configuration can be done with a variables
file as well as with environment variables. The configuration
file could be something like:

```bash
stack_name = "my-k8s"
ami_distro = "ubuntu"
authorized_keys = [
  "ssh-rsa AAAAB3NzaC1yc2E...",
]
aws_region = "eu-central-1"
aws_az = "eu-central-1a"
aws_access_key = "AKIAZKQ2..."
aws_secret_key = "ORdkX3vw..."
```

or you could provide these values in environmenta variables
when launching `terraform`:

```bash
$ TF_VAR_stack_name="my-k8s" \
  TF_VAR_aws_access_key="AKIAZKQ2..." \
  TF_VAR_aws_secret_key="ORdkX3vw..." \
  terraform apply -auto-approve
```

You could be intersested in customizing some variables like:

* `stack_name`: identifier to make all your resources unique and avoid
clashes with other users of this Terraform project.
* `authorized_keys`: a list of `ssh` public key to populate in the machines, 
used for accessing all the nodes. The private key must have been added to
the ssh agent and the agenet must be running.
* `ami_distro`: the Linux distro to use, currently `ubuntu` or `fedora`.
* `aws_region`: name of the region to be used.
* `aws_az`: the AWS Availability Zone
* `vpc_cidr`, `vpc_cidr_public`, `vpc_cidr_private`: the subnet CIDRs
for the VPC and the public and private subnets.
* `master_size` and `worker_size`: the VM size for the masters
and workers.

## Topology

The cluster will be made by these machines:

  * a Load Balancer that redirects requests to the masters.
  * a bastion host, used for accessing the machines though `ssh`,
  with a public IP and port 22 open.
  * `var.masters` master nodes (by default, two), not accessible
  from the outside.
  * `var.workers` worker nodes (by default, two), not accessible
  from the outside.

## Some notes on the Terraform code

There are some constraints imposed by Terraform/AWS that must be
taken into account when using this code as a base for your own
deployments. 

### Cluster creation order

Firstly, the cluster creation order in AWS is not _natural_
for using our provisioner. The problem comes when `kubeadm` is started
in the seeder as part of the provisioning. At some point it tries to
access the API server through the Load Balancer. However, the Load
Balancer creation depends on the very same instance being fully
created _and provisioned_, so there is a deadlock in the
creation process.

The only solution is to do the `kubeadm` provisioning _after_
the instance and the Load Balancer have been created, using a
`null_resource` where the `provisioner "kubeadm"` is embeded.

### Nodenames

Nodes must be registered with the private DNS names provided
by AWS. This can be accomplished by using the `private_dns` name
as the `nodename` in the _provisioner_.

### Autoscaling groups

_TODO_

 
