#####################
# Cluster variables #
#####################

variable "img" {
  type        = "string"
  default     = "lxd-kubeadm"
  description = "image name"
}

variable "distrobuilder" {
  type        = "string"
  default     = "distrobuilder-opensuse.yaml"
  description = "image name"
}

variable "force_img" {
  type        = "string"
  default     = ""
  description = "force the image re-creation"
}

variable "master_count" {
  default     = 1
  description = "Number of masters to be created"
}

variable "worker_count" {
  default     = 1
  description = "Number of workers to be created"
}

variable "kubeconfig" {
  default     = "kubeconfig.local"
  description = "Local kubeconfig file"
}

variable "name_prefix" {
  type        = "string"
  default     = "kubeadm-"
  description = "Optional prefix to be able to have multiple clusters on one host"
}

variable "private_key" {
  type        = "string"
  default     = "~/.ssh/id_rsa"
  description = "filename of ssh private key used for accessing all the nodes. a corresponding .pub file must exist"
}

variable "ssh_user" {
  type        = "string"
  default     = "root"
  description = "The SSH user"
}

variable "ssh_pass" {
  type        = "string"
  default     = "linux"
  description = "The SSH password"
}

variable "domain_name" {
  type        = "string"
  default     = "test.net"
  description = "The domain name"
}

variable "manifests" {
  type = "list"
  default = []
  description = "List of manifests to load after setting up the first master"
}
