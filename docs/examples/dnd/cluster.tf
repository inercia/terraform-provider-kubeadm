provider "docker" {
  version = "~> 2.0.0"
  host    = "${var.daemon}"
}

locals {
  gateway_ip = "${cidrhost(var.nodes_network, 1)}"
  haproxy_ip = "${cidrhost(var.nodes_network, 2)}"
  cache_ip   = "${cidrhost(var.nodes_network, 3)}"
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
# docker registry cache
##########################

# start a docker registry cache
# https://github.com/rpardini/docker-registry-proxy
resource "docker_container" "cache" {
  name                  = "${var.name_prefix}cache"
  image                 = "rpardini/docker-registry-proxy:0.2.4"
  hostname              = "${var.name_prefix}cache"
  start                 = true
  privileged            = true
  must_run              = true
  restart               = "no"
  destroy_grace_seconds = 60
  network_mode          = "bridge"
  domainname            = "${var.domain_name}"

  ports {
    internal = 3128
    external = 3128
  }

  networks_advanced {
    name         = "${docker_network.network.name}"
    ipv4_address = "${local.cache_ip}"
  }

  host {
    host = "cache.local"
    ip   = "${local.cache_ip}"
  }

  labels {
    type        = "cache"
    environment = "${var.name_prefix}"
  }

  volumes {
    host_path      = "${path.cwd}/docker_mirror_cache"
    container_path = "/docker_mirror_cache"
  }

  volumes {
    host_path      = "${path.cwd}/docker_mirror_certs"
    container_path = "/ca"
  }

  env = [
    "REGISTRIES=k8s.gcr.io gcr.io quay.io",
  ]
}

output "cache" {
  value = [
    "${docker_container.cache.ip_address}",
  ]
}

# will use this dropin in the masters/workers
data "template_file" "docker_dropin_cache" {
  template = <<EOF
[Service]
Environment="HTTP_PROXY=http://${docker_container.cache.ip_address}:3128/"
Environment="HTTPS_PROXY=http://${docker_container.cache.ip_address}:3128/"
EOF
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
# this is not completely safe, as we start the haproxy before all the masters are up,
# so `kubeadm init` can run the probe in the wrong backend.
resource "docker_container" "haproxy" {
  name                  = "${var.name_prefix}haproxy"
  image                 = "haproxy"
  hostname              = "${var.name_prefix}haproxy"
  start                 = true
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
    ip   = "${local.haproxy_ip}"
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
    plugin = "${var.cni}"

    # note: OpenSUSE images use a non-standard directory
    bin_dir = "/usr/lib/cni"
  }

  helm {
    install = true
  }

  dashboard {
    install = true
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
    "docker_container.cache",
    "null_resource.base_image",
  ]

  labels {
    type        = "master"
    environment = "${var.name_prefix}"
  }

  host {
    host = "haproxy.local"
    ip   = "${local.haproxy_ip}"
  }

  upload {
    content = "${data.template_file.docker_dropin_cache.rendered}"
    file    = "/etc/systemd/system/docker.service.d/http-proxy.conf"
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

  # setup the Docker registry cache
  provisioner "remote-exec" {
    inline = [
      # Get the CA certificate from the proxy and make it a trusted root.
      # copy it to the OS-specifi directory: for OpenSUSE: /etc/pki/trust/anchors
      "mkdir -p /etc/pki/trust/anchors/",

      "curl http://${docker_container.cache.ip_address}:3128/ca.crt > /etc/pki/trust/anchors/docker_registry_proxy.crt",
      "echo 'docker_registry_proxy.crt' >> /etc/ca-certificates.conf",
      "update-ca-certificates --fresh",

      # Reload systemd
      "systemctl daemon-reload",

      # Restart dockerd
      "systemctl restart docker.service",
    ]
  }

  provisioner "kubeadm" {
    config    = "${kubeadm.main.config}"
    role      = "master"
    join      = "${count.index == 0 ? "" : docker_container.master.0.ip_address}"
    manifests = "${var.manifests}"

    ignore_checks = [
      "KubeletVersion",  // the kubelet version in the base image can be very different
    ]
  }

  # provisioner for removing the node from the cluster
  provisioner "kubeadm" {
    when   = "destroy"
    config = "${kubeadm.main.config}"
    drain  = true
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
    "docker_container.cache",
    "docker_container.master",
  ]

  host {
    host = "haproxy.local"
    ip   = "${local.haproxy_ip}"
  }

  labels {
    type        = "worker"
    environment = "${var.name_prefix}"
  }

  upload {
    content = "${data.template_file.docker_dropin_cache.rendered}"
    file    = "/etc/systemd/system/docker.service.d/http-proxy.conf"
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

  # setup the Docker registry cache
  provisioner "remote-exec" {
    inline = [
      # Get the CA certificate from the proxy and make it a trusted root.
      # copy it to the OS-specifi directory: for OpenSUSE: /etc/pki/trust/anchors
      "mkdir -p /etc/pki/trust/anchors/",

      "curl http://${docker_container.cache.ip_address}:3128/ca.crt > /etc/pki/trust/anchors/docker_registry_proxy.crt",
      "echo 'docker_registry_proxy.crt' >> /etc/ca-certificates.conf",
      "update-ca-certificates --fresh",

      # Reload systemd
      "systemctl daemon-reload",

      # Restart dockerd
      "systemctl restart docker.service",
    ]
  }

  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    join   = "${lookup(docker_container.master.0.network_data[0], "ip_address")}"
    role   = "worker"

    ignore_checks = [
      "KubeletVersion",  // the kubelet version in the base image can be very different
    ]
  }

  # provisioner for removing the node from the cluster
  provisioner "kubeadm" {
    when   = "destroy"
    config = "${kubeadm.main.config}"
    drain  = true
  }

  lifecycle {
    create_before_destroy = true
  }
}

output "workers" {
  value = "${docker_container.worker.*.ip_address}"
}
