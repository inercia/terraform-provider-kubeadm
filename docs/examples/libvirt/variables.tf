#####################
# variables
#####################

variable "master_count" {
  default = "1"
}

variable "worker_count" {
  default = "1"
}

variable "cni" {
  default     = "flannel"
  description = "CNI driver"
}

variable "ssh" {
  default = "../ssh/id_rsa"
}

variable "image" {
  default = "https://cloud-images.ubuntu.com/releases/xenial/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img"
}

variable "image_pool" {
  default = "default"
}

variable "name_prefix" {
  type        = "string"
  default     = "kadm-lv-"
  description = "Optional prefix to be able to have multiple clusters on one host"
}

variable "manifests" {
  type = "list"

  default = [
    "https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml",
    "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/mandatory.yaml",
    "https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/provider/cloud-generic.yaml",
    "https://raw.githubusercontent.com/kubernetes/dashboard/master/aio/deploy/recommended/kubernetes-dashboard.yaml",
  ]

  description = "List of manifests to load after setting up the first master"
}

variable "kubeconfig" {
  default     = "kubeconfig.local"
  description = "Local kubeconfig file"
}