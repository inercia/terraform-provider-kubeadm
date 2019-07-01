#####################
# global
#####################

provider "libvirt" {
  uri = "qemu:///system"
}

resource "libvirt_network" "backend" {
  name      = "${var.name_prefix}net"
  mode      = "nat"
  domain    = "local"
  addresses = ["10.17.6.0/24"]
}

resource "libvirt_volume" "base" {
  name   = "${var.name_prefix}base.img"
  source = "${var.image}"
  pool   = "${var.image_pool}"
}

data "template_file" "cloud_init_user_data" {
  template = "${file("cloud-init/user-data.cfg.tpl")}"
}

##########################
# Kubeadm #
##########################

resource "kubeadm" "main" {
  network {
    dns_domain = "mycluster.com"
    services   = "10.25.0.0/16"
  }

  runtime {
    # note: "crio" seems to have some issues: some pods keep erroring
    #       in "ContainerCreating", with "failed to get network status for pod sandbox"
    #       switching to Docker solves those problems...
    engine = "docker"
  }
}

#####################
# kube-master
#####################
resource "libvirt_volume" "master_volume" {
  name           = "${var.name_prefix}master.img"
  pool           = "${var.image_pool}"
  base_volume_id = "${libvirt_volume.base.id}"
  size           = "10737418240"
}

resource "libvirt_cloudinit_disk" "ci" {
  name      = "${var.name_prefix}ci.iso"
  pool      = "${var.image_pool}"
  user_data = "${data.template_file.cloud_init_user_data.rendered}"
}

resource "libvirt_domain" "master" {
  name      = "${var.name_prefix}master"
  memory    = 512
  cloudinit = "${libvirt_cloudinit_disk.ci.id}"

  disk {
    volume_id = "${libvirt_volume.master_volume.id}"
  }

  connection {
    type     = "ssh"
    user     = "root"
    password = "linux"
  }

  network_interface {
    network_id     = "${libvirt_network.backend.id}"
    hostname       = "${var.name_prefix}master.local"
    wait_for_lease = 1
  }

  provisioner "kubeadm" {
    config     = "${kubeadm.main.config}"
    kubeconfig = "${var.kubeconfig}"
    manifests  = "${var.manifests}"
  }
}

output "masters" {
  value = ["${libvirt_domain.master.*.network_interface.0.addresses}"]
}

#####################
# kube-minion
#####################
resource "libvirt_volume" "minion_volume" {
  count          = "${var.minions}"
  name           = "${var.name_prefix}minion${count.index}.img"
  pool           = "${var.image_pool}"
  base_volume_id = "${libvirt_volume.base.id}"
  size           = "10737418240"
}

resource "libvirt_domain" "minion" {
  count      = "${var.minions}"
  name       = "${var.name_prefix}minion${count.index}"
  depends_on = ["libvirt_domain.master"]
  memory     = 512
  cloudinit  = "${libvirt_cloudinit_disk.ci.id}"

  disk {
    volume_id = "${element(libvirt_volume.minion_volume.*.id, count.index)}"
  }

  connection {
    type     = "ssh"
    user     = "root"
    password = "linux"
  }

  network_interface {
    network_id     = "${libvirt_network.backend.id}"
    hostname       = "${var.name_prefix}minion${count.index}.local"
    wait_for_lease = 1
  }

  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    join   = "${libvirt_domain.master.network_interface.0.addresses.0}"

    #ignore_checks = [
    #  "NumCPU",
    #  "FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
    #  "Swap",
    #]
  }
}

output "workers" {
  value = ["${libvirt_domain.minion.*.network_interface.0.addresses}"]
}
