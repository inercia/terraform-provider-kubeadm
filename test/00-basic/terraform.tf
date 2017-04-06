#####################
# variables
#####################

variable "minions" { default = "1" }

variable "ssh" { default = "../ssh/id_rsa" }

variable "image" { default = "Base-openSUSE-Leap-42.2.x86_64-cloud_ext4.qcow2" }

variable "image_pool" { default = "personal" }

#####################
# global
#####################

provider "libvirt" {
  uri = "qemu:///system"
}

resource "libvirt_network" "backend" {
  name      = "k8snet"
  mode      = "nat"
  domain    = "k8s.local"
  addresses = ["10.17.6.0/24"]
}

resource "libvirt_volume" "base_volume" {
  name             = "k8s_base.img"
  pool             = "${var.image_pool}"
  base_volume_name = "${var.image}"
}

resource "kubeadm" "main" {
  name = "my_cluster"
  services_cidr = "10.25.0.0/16"
}

#####################
# kube-master
#####################
resource "libvirt_volume" "master_volume" {
  name           = "k8s_master.img"
  pool           = "${var.image_pool}"
  base_volume_id = "${libvirt_volume.base_volume.id}"
}

resource "libvirt_cloudinit" "master_ci" {
  name               = "k8s_master_ci.iso"
  local_hostname     = "master.k8s.local"
  ssh_authorized_key = "${file("../ssh/id_rsa.pub")}"
  pool               = "${var.image_pool}"
}

resource "libvirt_domain" "master" {
  name       = "k8s_master"
  memory     = 512
  cloudinit  = "${libvirt_cloudinit.master_ci.id}"

  disk {
    volume_id = "${libvirt_volume.master_volume.id}"
  }

  connection {
    type        = "ssh"
    user        = "root"
    private_key = "${file(var.ssh)}"
  }

  network_interface {
    network_id     = "${libvirt_network.backend.id}"
    hostname       = "master.k8s.local"
    wait_for_lease = 1
  }

  provisioner "kubeadm" {
    config      = "${kubeadm.main.config.master}"
  }
}

#####################
# kube-minion
#####################
resource "libvirt_volume" "minion_volume" {
  count          = "${var.minions}"
  name           = "k8s_minion${count.index}.img"
  pool           = "${var.image_pool}"
  base_volume_id = "${libvirt_volume.base_volume.id}"
}

resource "libvirt_cloudinit" "minion_ci" {
  count              = "${var.minions}"
  name               = "k8s_minion${count.index}_ci.iso"
  local_hostname     = "minion${count.index}.k8s.local"
  ssh_authorized_key = "${file("../ssh/id_rsa.pub")}"
  pool               = "${var.image_pool}"
}

resource "libvirt_domain" "minion" {
  count      = "${var.minions}"
  name       = "k8s_minion${count.index}"
  depends_on = ["libvirt_domain.master"]
  memory     = 512
  cloudinit  = "${element(libvirt_cloudinit.minion_ci.*.id, count.index)}"

  disk {
    volume_id = "${element(libvirt_volume.minion_volume.*.id, count.index)}"
  }

  connection {
    type        = "ssh"
    user        = "root"
    private_key = "${file(var.ssh)}"
  }

  network_interface {
    network_id     = "${libvirt_network.backend.id}"
    hostname       = "minion${count.index}.k8s.local"
    wait_for_lease = 1
  }

  provisioner "kubeadm" {
    master      = "${libvirt_domain.master.network_interface.0.addresses.0}"
    config      = "${kubeadm.main.config.node}"
  }
}
