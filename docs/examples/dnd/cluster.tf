provider "docker" {
  host = "${var.daemon}"
}

##########################
# Kubeadm #
##########################

data "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  network {
    dns_domain = "mycluster.com"
    services   = "10.25.0.0/16"
  }

  runtime {
    engine = "docker"
  }

  cni {
    plugin = "flannel"

    # OpenSUSE images use a non-standard directory
    bin_dir = "/usr/lib/cni"
  }

  addons {
    helm      = "true"
    dashboard = "true"
  }
}

##########################
# Base image and profile #
##########################

resource "null_resource" "base_image" {
  provisioner "local-exec" {
    command     = "make IMAGE='${var.img}'"
    interpreter = ["bash", "-c"]
  }
}

resource "docker_network" "network" {
  name            = "${var.name_prefix}net"
  check_duplicate = "true"
}

#####################
### Cluster masters #
#####################

resource "docker_container" "master" {
  count                 = "${var.master_count}"
  name                  = "${var.name_prefix}master-${count.index}"
  image                 = "${var.img}"
  hostname              = "${var.name_prefix}master-${count.index}"
  start                 = true
  rm                    = true
  privileged            = true
  must_run              = true
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  networks              = ["${docker_network.network.name}"]
  depends_on            = ["null_resource.base_image"]

  labels {
    type        = "master"
    environment = "${var.name_prefix}"
  }

  # API server
  ports {
    external = "6443"
    internal = 6443
  }

  ports {
    external = "1080"
    internal = 80
  }

  volumes {
    host_path      = "/sys/fs/cgroup"
    container_path = "/sys/fs/cgroup"
    read_only      = "true"
  }

  connection {
    type     = "ssh"
    host     = "${self.ip_address}"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "kubeadm" {
    config    = "${data.kubeadm.main.config}"
    manifests = "${var.manifests}"

    # ignore_checks = [
    #   "NumCPU",
    #   "FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
    #   "Swap",
    # ]
  }

  lifecycle {
    create_before_destroy = true

    # ignore_changes = ["associate_public_ip_address"]
  }
}

output "masters" {
  value = ["${docker_container.master.*.ip_address}"]
}

####################
## Cluster workers #
####################

resource "docker_container" "worker" {
  count                 = "${var.worker_count}"
  name                  = "${var.name_prefix}worker-${count.index}"
  image                 = "${var.img}"
  hostname              = "${var.name_prefix}worker-${count.index}"
  start                 = true
  rm                    = true
  privileged            = true
  must_run              = true
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  networks              = ["${docker_network.network.name}"]
  depends_on            = ["docker_container.master"]

  labels {
    type        = "worker"
    environment = "${var.name_prefix}"
  }

  volumes {
    host_path      = "/sys/fs/cgroup"
    container_path = "/sys/fs/cgroup"
    read_only      = "true"
  }

  connection {
    type     = "ssh"
    host     = "${self.ip_address}"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"
    join   = "${lookup(docker_container.master.0.network_data[0], "ip_address")}"
  }

  lifecycle {
    create_before_destroy = true

    # ignore_changes = ["associate_public_ip_address"]
  }
}

output "workers" {
  value = "${docker_container.worker.*.ip_address}"
}
