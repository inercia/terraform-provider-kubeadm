provider "docker" {
  # Travis-CI runs an "old" version of Docker
  # so we must force a version that uses a compatible API
  # see https://releases.hashicorp.com/terraform-provider-docker/
  # see https://www.terraform.io/docs/configuration/providers.html#version-provider-versions
  # version = "~> 1.2.0"

  host = "${var.daemon}"
}

locals {
  gateway_ip = "${cidrhost(var.nodes_network, 1)}"
  haproxy_ip = "${cidrhost(var.nodes_network, 2)}"
}

resource "docker_network" "network" {
  name            = "${var.name_prefix}net"
  check_duplicate = "true"

  ipam_config {
    subnet  = "${var.nodes_network}"
    gateway = "${local.gateway_ip}"
  }
}

##########################
# haproxy
##########################

# generate one haproxy backend line per master
data "template_file" "haproxy_backends" {
  count = "${var.master_count}"
  template = <<EOF
  server $${fqdn} $${ip}:6443 check check-ssl verify none
EOF

  vars = {
    fqdn = "${var.name_prefix}master-${count.index}.${var.domain_name}"
    ip   = "${cidrhost(var.nodes_network, 16 + count.index)}"
  }
}

data "template_file" "haproxy_config" {
  template = <<EOF
    defaults
      timeout connect 10s
      timeout client 86400s
      timeout server 86400s

    frontend apiserver
      bind :6443
      default_backend apiserver-backend

    backend apiserver-backend
      option httpchk GET /healthz
      $${backends}
EOF

  vars = {
    backends = "${join("      ", data.template_file.haproxy_backends.*.rendered)}"
  }
}

# start a haproxy instance as a load balancer for all the masters
resource "docker_container" "haproxy" {
  name                  = "${var.name_prefix}haproxy"
  image                 = "haproxy"
  hostname              = "${var.name_prefix}haproxy"
  start                 = true
  rm                    = true
  privileged            = true
  must_run              = true
  restart               = "no"
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  domainname            = "${var.domain_name}"

  networks_advanced {
    name         = "${docker_network.network.name}"
    ipv4_address = "${local.haproxy_ip}"
  }

  host {
    host = "haproxy.local"
    ip = "${local.haproxy_ip}"
  }

  labels {
    type        = "haproxy"
    environment = "${var.name_prefix}"
  }

  upload {
    content = "${data.template_file.haproxy_config.rendered}"
    file    = "/usr/local/etc/haproxy/haproxy.cfg"
  }

  volumes {
    host_path      = "/sys/fs/cgroup"
    container_path = "/sys/fs/cgroup"
    read_only      = "true"
  }
}

output "lb" {
  value = [
    "${docker_container.haproxy.ip_address}",
  ]
}


##########################
# Kubeadm #
##########################

resource "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  network {
    dns_domain = "${var.domain_name}.k8s.local"
    services   = "10.25.0.0/16"
  }

  api {
    //  use the load balancer as an external IP
    external = "${local.haproxy_ip}"
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
    command = "make IMAGE='${var.img}'"

    interpreter = [
      "bash",
      "-c",
    ]
  }
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
  restart               = "no"
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  domainname            = "${var.domain_name}"

  networks_advanced {
    name         = "${docker_network.network.name}"
    ipv4_address = "${cidrhost(var.nodes_network, 16 + count.index)}"
  }

  depends_on = [
    "docker_container.haproxy",
    "null_resource.base_image",
  ]

  labels {
    type        = "master"
    environment = "${var.name_prefix}"
  }

  host {
    host = "haproxy.local"
    ip = "${local.haproxy_ip}"
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
    config    = "${kubeadm.main.config}"
    role      = "master"
    join      = "${count.index == 0 ? "" : docker_container.master.0.ip_address}"
    manifests = "${var.manifests}"
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "masters" {
  value = [
    "${docker_container.master.*.ip_address}",
  ]
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
  restart               = "no"
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  domainname            = "${var.domain_name}"

  networks_advanced {
    name         = "${docker_network.network.name}"
    ipv4_address = "${cidrhost(var.nodes_network, 32 + count.index)}"
  }

  depends_on = [
    "docker_container.haproxy",
    "docker_container.master",
  ]

  host {
    host = "haproxy.local"
    ip = "${local.haproxy_ip}"
  }

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
    config = "${kubeadm.main.config}"
    join   = "${lookup(docker_container.master.0.network_data[0], "ip_address")}"
    role   = "worker"
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "workers" {
  value = "${docker_container.worker.*.ip_address}"
}
