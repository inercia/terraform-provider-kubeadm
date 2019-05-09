variable "stack_name" {
  default     = "k8s-test"
  description = "identifier to make all your resources unique and avoid clashes with other users of this terraform project"
}

variable "region" {
  default     = "eu-west-3"
  description = "Name of the region to be used - London by default"
}

variable "ami_name_pattern" {
  default     = ".*ubuntu-bionic.*server.*"
  description = "Pattern for choosing the AMI image"
}

variable "ami_owner" {
  default     = "099720109477"
  description = "AMI owner id"
}

variable "kubeconfig" {
  default     = "kubeconfig.local"
  description = "A local copy of the admin kubeconfig created after the cluster initialization"
}

variable "private_key" {
  type        = "string"
  default     = "~/.ssh/id_rsa"
  description = "filename of ssh private key used for accessing all the nodes. the equivalent .pub file must exist"
}

variable "subnet_cidr" {
  type        = "string"
  default     = "10.0.0.0/16"
  description = "Subnet CIDR"
}

variable "access_key" {
  default     = ""
  description = "AWS access key"
}

variable "secret_key" {
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

variable "public_worker" {
  description = "Weither or not the workers should have a public IP"
  default     = true
}

variable "tags" {
  type        = "map"
  default     = {}
  description = "Extra tags used for the AWS resources created"
}
