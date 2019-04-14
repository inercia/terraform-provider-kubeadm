#######################
# Cluster declaration #
#######################

provider "lxd" {
  generate_client_certificates = true
  accept_remote_certificate    = true
}


data "template_file" "authorized_keys_line" {
  count    = "${length(var.authorized_keys)}"
  template = "${file(element(var.authorized_keys, count.index))}"
}

data "template_file" "authorized_keys_file_contents" {
  template = "${join("\n", data.template_file.authorized_keys_line.*.rendered)}"
}

locals {
  authorized_keys_file = <<EOT
${data.template_file.authorized_keys_file_contents.rendered}
${file("~/.ssh/id_dsa.pub")}
EOT
}


##########################
# Kubeadm #
##########################

data "kubeadm" "main" {
  api {
    external = "loadbalancer.external.com"
  }

  network {
    dns_domain = "my_cluster.local"
    services = "10.25.0.0/16"
  }
}

##########################
# Base image and profile #
##########################

resource "null_resource" "base_image" {
  # make sure we have an opensuse-caasp image
  # if that is not the case, build one with the help of distrobuilder
  provisioner "local-exec" {
    command = "./support/build-image.sh --img '${var.img}' --force '${var.force_img}'"
  }
}

# from https://github.com/juju-solutions/bundle-canonical-kubernetes/issues/566#issuecomment-386195937
resource "lxd_profile" "k8s" {
  name = "k8s"

  device {
    name = "root"
    type = "disk"

    properties {
      pool = "default"
      path = "/"
    }
  }

  device {
    name = "default"   # must match the `lxc storage show default` used
    type = "unix-block"

    properties {
      source = "/dev/sda3"
      path = "/dev/sda3"
      #source = "/var/snap/lxd/common/lxd/storage-pools/default"
      #path = "/var/snap/lxd/common/lxd/storage-pools/default"
    }
  }

  device {
    name = "aadisable1"
    type = "disk"

    properties {
      source = "/dev/null"
      path = "/sys/module/nf_conntrack/parameters/hashsize"
    }
  }

  device {
    name = "aadisable2"
    type = "disk"

    properties {
      source = "/dev/null"
      path = "/sys/module/apparmor/parameters/enabled"
    }
  }

  config {
    limits.cpu = 1
    limits.cpu.allowance = "25%"
    linux.kernel_modules = "ip_tables,ip6_tables,netlink_diag,nf_nat,overlay"
    security.privileged = "true"
    security.nesting = "true"
    raw.lxc = <<EOF
lxc.apparmor.profile = unconfined
lxc.cap.drop =
lxc.cgroup.devices.allow = a
lxc.mount.auto = proc:rw sys:rw
EOF
  }
}

#####################
### Cluster masters #
#####################

resource "lxd_container" "master" {
  count      = "${var.master_count}"
  name       = "${var.name_prefix}master-${count.index}"
  image      = "${var.img}"
  depends_on = ["null_resource.base_image"]
  profiles   = ["default", "${lxd_profile.k8s.name}"]

  connection {
    type     = "ssh"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "file" {
    content     = "${local.authorized_keys_file}"
    destination = "/root/.ssh/authorized_keys"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config.init}"
  }
}

output "masters" {
  value = ["${lxd_container.master.*.ip_address}"]
}

####################
## Cluster workers #
####################

resource "lxd_container" "worker" {
  count      = "${var.worker_count}"
  name       = "${var.name_prefix}worker-${count.index}"
  image      = "${var.img}"
  depends_on = ["null_resource.base_image"]
  profiles   = ["default", "${lxd_profile.k8s.name}"]

  connection {
    type     = "ssh"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "file" {
    content     = "${local.authorized_keys_file}"
    destination = "/root/.ssh/authorized_keys"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config.join}"
    join = "${lxd_container.master.0.ip_address}"
  }
}

output "workers" {
  value = ["${lxd_container.worker.*.ip_address}"]
}

######################
# Load Balancer node #
######################
data "template_file" "haproxy_backends_master" {
  count    = "${var.master_count}"
  template = "${file("templates/haproxy-backends.tpl")}"

  vars = {
    fqdn = "${var.name_prefix}master-${count.index}.${var.name_prefix}${var.domain_name}"
    ip   = "${element(lxd_container.master.*.ip_address, count.index)}"
  }
}

data "template_file" "haproxy_cfg" {
  template = "${file("templates/haproxy.cfg.tpl")}"

  vars = {
    backends = "${join("      ", data.template_file.haproxy_backends_master.*.rendered)}"
  }
}

resource "lxd_container" "lb" {
  name       = "${var.name_prefix}lb"
  image      = "${var.img}"
  depends_on = ["lxd_container.master"]

  connection {
    type     = "ssh"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "file" {
    content     = "${local.authorized_keys_file}"
    destination = "/root/.ssh/authorized_keys"
  }

  provisioner "file" {
    content     = "${data.template_file.haproxy_cfg.rendered}"
    destination = "/etc/haproxy/haproxy.cfg"
  }

  provisioner "remote-exec" {
    inline = "systemctl enable --now haproxy"
  }
}

output "ip_lb" {
  value = "${lxd_container.lb.ip_address}"
}
