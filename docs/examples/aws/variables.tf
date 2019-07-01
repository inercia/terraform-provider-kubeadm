variable "stack_name" {
  default     = "k8s-test"
  description = "identifier to make all your resources unique and avoid clashes with other users of this terraform project"
}

variable "aws_region" {
  default     = "eu-west-3"
  description = "Name of the region to be used"
}

variable "aws_az" {
  type        = "string"
  description = "AWS Availability Zone"
  default     = "eu-west-3a"
}

variable "ami_distro" {
  default     = "ubuntu"
  description = "AMI distro"
}

variable "kubeconfig" {
  default     = "kubeconfig.local"
  description = "A local copy of the admin kubeconfig created after the cluster initialization"
}

variable "vpc_cidr" {
  type        = "string"
  default     = "10.1.0.0/16"
  description = "Subnet CIDR"
}

variable "public_subnet" {
  type        = "string"
  description = "CIDR blocks for each public subnet of vpc"
  default     = "10.1.1.0/24"
}

variable "private_subnet" {
  type        = "string"
  description = "Private subnet of vpc"
  default     = "10.1.4.0/24"
}

variable "aws_access_key" {
  default     = ""
  description = "AWS access key"
}

variable "aws_secret_key" {
  default     = ""
  description = "AWS secret key"
}

variable "master_size" {
  default     = "t2.micro"
  description = "Size of the master nodes"
}

variable "masters" {
  default     = 2
  description = "Number of master nodes"
}

variable "worker_size" {
  default     = "t2.micro"
  description = "Size of the worker nodes"
}

variable "workers" {
  default     = 2
  description = "Number of worker nodes"
}

variable "tags" {
  type        = "map"
  default     = {}
  description = "Extra tags used for the AWS resources created"
}

variable "authorized_keys" {
  type        = "list"
  default     = []
  description = "ssh keys to inject into all the nodes. First key will be used for creating a keypair."
}