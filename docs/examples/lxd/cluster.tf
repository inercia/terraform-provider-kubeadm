#######################
# Cluster declaration #
#######################

provider "lxd" {
  generate_client_certificates = true
  accept_remote_certificate    = true
}

##########################
# Kubeadm #
##########################

resource "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  network {
    dns_domain = "mycluster.com"
    services   = "10.25.0.0/16"
  }

  runtime {
    # note: "crio" seems to have some issues in LXD: some pods keep erroring
    #       in "ContainerCreating", with "failed to get network status for pod sandbox"
    #       switching to Docker solves those problems...
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

# auto-detect the root device, so it can be mounted in the LXC container
# and the kubelet can be happy (it wants to detect how much free
# space is left and so on...)
data "external" "root_device" {
  program = [
    "sh",
    "./support/get-root-device.sh",
  ]
}

resource "null_resource" "base_image" {
  # make sure we have an opensuse-caasp image
  # if that is not the case, build one with the help of distrobuilder
  provisioner "local-exec" {
    command     = "./images/build-image.sh --img '${var.img}' --yaml './images/${var.distrobuilder}' --force '${var.force_img}'"
    interpreter = ["bash", "-c"]
  }
}

# see https://github.com/corneliusweig/kubernetes-lxd
resource "lxd_profile" "kubelet" {
  name = "kubelet"

  device {
    name = "root"
    type = "disk"

    properties {
      pool = "default"
      path = "/"
    }
  }

  device {
    name = "default"
    type = "unix-block"

    properties {
      path   = "${data.external.root_device.result.device}"
    }
  }

  # map /sys/module/apparmor/parameters/enabled -> /dev/null
  device {
    name = "aadisable2"
    type = "disk"

    properties {
      source = "/dev/null"
      path   = "/sys/module/apparmor/parameters/enabled"
    }
  }

  # map /lib/modules -> /lib/modules
  device {
    name = "lib-modules"
    type = "disk"

    properties {
      source = "/lib/modules"
      path   = "/lib/modules"
    }
  }

  # map /dev/kvm -> /dev/null
  # device {
  #   name = "kvm"
  #   type = "unix-char"
  #
  #   properties {
  #     path   = "/dev/kvm"
  #   }
  # }

  device {
    name = "mem"
    type = "unix-char"

    properties {
      path   = "/dev/mem"
    }
  }

  config {
    limits.cpu           = 1
    limits.cpu.allowance = "25%"
    limits.memory        = "3GB"
    limits.memory.swap   = "false"

    #  for a privileged container which may create nested cgroups
    security.privileged = "true"
    security.nesting    = "true"

    # depending on the kernel of your host system, you need to add
    # further kernel modules here. The ones listed above are for
    # networking and for dockers overlay filesystem.
    linux.kernel_modules = "br_netfilter,ip_tables,ip6_tables,ip_vs,ip_vs_rr,ip_vs_wrr,ip_vs_sh,netlink_diag,nf_nat,overlay,xt_conntrack"

    environment.http_proxy = ""
    user.network_mode      = ""

    raw.lxc = <<EOF
lxc.apparmor.profile=unconfined
lxc.cap.drop=
lxc.cgroup.devices.allow=a
lxc.mount.auto=proc:rw sys:rw
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
  profiles   = ["default", "${lxd_profile.kubelet.name}"]

  connection {
    type     = "ssh"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "file" {
    content     = "${file("${var.private_key}.pub")}"
    destination = "/root/.ssh/authorized_keys"
  }

  provisioner "kubeadm" {
    config     = "${kubeadm.main.config}"
    manifests  = "${var.manifests}"
    # ignore_checks = [
    #   "NumCPU",
    #   "FileContent--proc-sys-net-bridge-bridge-nf-call-iptables",
    #   "Swap",
    # ]
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
  depends_on = ["lxd_container.master"]
  profiles   = ["default", "${lxd_profile.kubelet.name}"]

  connection {
    type     = "ssh"
    user     = "${var.ssh_user}"
    password = "${var.ssh_pass}"
  }

  provisioner "file" {
    content     = "${file("${var.private_key}.pub")}"
    destination = "/root/.ssh/authorized_keys"
  }

  provisioner "kubeadm" {
    config = "${kubeadm.main.config}"
    join   = "${lxd_container.master.0.ip_address}"
  }
}

output "workers" {
  value = ["${lxd_container.worker.*.ip_address}"]
}
