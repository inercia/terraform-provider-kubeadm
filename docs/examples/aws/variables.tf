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

variable "private_key" {
  type        = "string"
  default     = "~/.ssh/id_rsa"
  description = "filename of ssh private key used for accessing all the nodes. a corresponding .pub file must exist"
}

variable "vpc_cidr" {
  type        = "string"
  default     = "10.0.0.0/16"
  description = "Subnet CIDR"
}

variable "vpc_cidr_public" {
  type        = "string"
  default     = "10.0.70.0/24"
  description = "Subnet CIDR for the public subnet"
}

variable "vpc_cidr_private" {
  type        = "string"
  default     = "10.0.80.0/24"
  description = "Subnet CIDR for the private subnet"
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
  default     = 1
  description = "Number of master nodes"
}

variable "worker_size" {
  default     = "t2.micro"
  description = "Size of the worker nodes"
}

variable "workers" {
  default     = 1
  description = "Number of worker nodes"
}

variable "tags" {
  type        = "map"
  default     = {}
  description = "Extra tags used for the AWS resources created"
}
