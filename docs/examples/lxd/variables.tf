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

variable "authorized_keys" {
  type        = "list"
  default     = []
  description = "ssh keys to inject into all the nodes"
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

  default = [
    "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml",
    "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/provider/cloud-generic.yaml",
    "https://raw.githubusercontent.com/kubernetes/dashboard/master/aio/deploy/recommended/kubernetes-dashboard.yaml",
  ]

  description = "List of manifests to load after setting up the first master"
}

variable "root_device" {
  type        = "string"
  default     = "/dev/sda3"                                                                   # must match the `lxc storage show default` used
  description = "The root device in the host (so the kubelet can see how much space is free)"
}
