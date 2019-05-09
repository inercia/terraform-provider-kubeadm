provider "aws" {
  region     = "${var.region}"
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
}

locals {
  tags = "${merge(
    map("Name", var.stack_name,
        "Environment", var.stack_name,
        format("kubernetes.io/cluster/%v", var.stack_name), "owned"),
    var.tags)}"
}

###########################################
# common
###########################################

data "template_file" "cloud-init" {
  template = "${file("cloud-init/cloud-init.yaml.tpl")}"

  vars {
    public_key = "${file("${var.private_key}.pub")}"
  }
}

resource "aws_key_pair" "keypair" {
  key_name   = "${var.stack_name}"
  public_key = "${file("${var.private_key}.pub")}"
}

###########################################
# load balancer
###########################################

resource "aws_alb" "lb" {
  name                       = "${var.stack_name}-kube-lb"
  internal                   = false
  load_balancer_type         = "network"
  enable_deletion_protection = false
  subnets                    = ["${aws_subnet.public.*.id}"]
  tags                       = "${merge(local.tags, map("Class", "LoadBalancer"))}"
}

resource "aws_alb_target_group" "masters" {
  name        = "${var.stack_name}-target-group-masters"
  port        = 6443
  protocol    = "TCP"
  vpc_id      = "${aws_vpc.main.id}"
  target_type = "ip"
}

resource "aws_alb_listener" "api_server" {
  load_balancer_arn = "${aws_alb.lb.arn}"
  port              = "6443"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_alb_target_group.masters.arn}"
  }
}

resource "aws_alb_target_group_attachment" "master" {
  count            = "${var.masters}"
  target_group_arn = "${aws_alb_target_group.masters.arn}"
  target_id        = "${element(aws_instance.master.*.id, count.index)}"
  port             = 6443
}

###########################################
# network
###########################################
resource "aws_vpc" "main" {
  cidr_block                       = "${var.subnet_cidr}"
  enable_dns_support               = true
  enable_dns_hostnames             = true
  assign_generated_ipv6_cidr_block = false
  tags                             = "${merge(local.tags, map("Class", "VPC"))}"
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.main.id}"
  tags   = "${merge(local.tags, map("Class", "GW"))}"
}

resource "aws_route_table" "r" {
  vpc_id = "${aws_vpc.main.id}"
  tags   = "${merge(local.tags, map("Class", "RouteTable"))}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }
}

resource "aws_subnet" "public" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "${cidrsubnet(var.subnet_cidr, 8, count.index)}"

  # map_public_ip_on_launch = true
  depends_on = ["aws_internet_gateway.gw"]
  tags       = "${merge(local.tags, map("Name","${var.stack_name}-subnet", "Class", "Subnet"))}"
}

resource "aws_route_table_association" "public" {
  subnet_id      = "${aws_subnet.public.id}"
  route_table_id = "${aws_route_table.r.id}"
}

###########################################
# network security
###########################################
resource "aws_security_group" "kubernetes" {
  name        = "${var.stack_name}"
  description = "Security rules for Kubernetes"
  vpc_id      = "${aws_vpc.main.id}"
  tags        = "${merge(local.tags, map("Class", "SecGroup"))}"
}

resource "aws_security_group_rule" "allow_all_from_self" {
  type                     = "ingress"
  from_port                = 0
  to_port                  = 0
  protocol                 = "-1"
  source_security_group_id = "${aws_security_group.kubernetes.id}"
  security_group_id        = "${aws_security_group.kubernetes.id}"
}

resource "aws_security_group_rule" "allow_ssh_from_anywhere" {
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.kubernetes.id}"
}

resource "aws_security_group_rule" "allow_k8s_from_admin" {
  type              = "ingress"
  from_port         = 6443
  to_port           = 6443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.kubernetes.id}"
}

resource "aws_security_group_rule" "allow_https_from_web" {
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.kubernetes.id}"
}

resource "aws_security_group_rule" "allow_http_from_web" {
  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.kubernetes.id}"
}

resource "aws_security_group_rule" "allow_all_out" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = "${aws_security_group.kubernetes.id}"
}

###########################################
# images
###########################################

data "aws_ami" "latest_ami" {
  name_regex  = "${var.ami_name_pattern}"
  most_recent = true
  owners      = ["${var.ami_owner}"]

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

###########################################
# Kubeadm #
###########################################

data "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  api {
    external = "${aws_alb.lb.dns_name}"
  }

  network {
    dns_domain = "k8s.local"
    services   = "10.25.0.0/16"
  }

  runtime {
    engine = "docker"
  }

  cni {
    plugin = "flannel"
  }

  addons {
    helm      = "true"
    dashboard = "true"
  }
}

###########################################
# masters
###########################################

resource "aws_instance" "master" {
  count                       = "${var.masters}"
  ami                         = "${data.aws_ami.latest_ami.id}"
  instance_type               = "${var.master_size}"
  subnet_id                   = "${element(aws_subnet.public.*.id, count.index)}"
  user_data                   = "${data.template_file.cloud-init.rendered}"
  vpc_security_group_ids      = ["${aws_security_group.kubernetes.id}"]
  associate_public_ip_address = true
  key_name                    = "${aws_key_pair.keypair.key_name}"
  tags                        = "${merge(local.tags, map("Name", format("%s-master-%d", var.stack_name, count.index), "Class", "Instance"))}"

  lifecycle {
    ignore_changes = [
      "ami",
      "user_data",
      "associate_public_ip_address",
    ]
  }

  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = "${file(var.private_key)}"
    host        = "${aws_instance.master.public_ip}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot prooperly detect a valid hostname
    nodename = "${aws_instance.master.private_dns}"

    install {
      auto = true
    }
  }
}

###########################################
# workers
###########################################
resource "aws_instance" "worker" {
  count                       = "${var.workers}"
  ami                         = "${data.aws_ami.latest_ami.id}"
  instance_type               = "${var.worker_size}"
  subnet_id                   = "${element(aws_subnet.public.*.id, count.index)}"
  user_data                   = "${data.template_file.cloud-init.rendered}"
  vpc_security_group_ids      = ["${aws_security_group.kubernetes.id}"]
  associate_public_ip_address = false
  tags                        = "${merge(local.tags, map("Name", format("%s-worker-%d", var.stack_name, count.index), "Class", "Instance"))}"

  lifecycle {
    ignore_changes = [
      "ami",
      "user_data",
      "associate_public_ip_address",
    ]
  }

  connection {
    type         = "ssh"
    user         = "ubuntu"
    private_key  = "${file(var.private_key)}"
    host         = "${aws_instance.worker.private_ip}"
    bastion_host = "${aws_instance.master.0.public_ip}"
  }

  provisioner "kubeadm" {
    config   = "${data.kubeadm.main.config}"
    join     = "${element(aws_instance.master.*.private_ip, 0)}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot prooperly detect a valid hostname
    nodename = "${aws_instance.worker.private_dns}"

    install {
      auto = true
    }
  }
}

###########################################
# output
###########################################
output "ip_masters" {
  value = ["${aws_instance.master.*.public_ip}"]
}

output "ip_workers" {
  value = ["${aws_instance.worker.*.public_ip}"]
}

output "lb_dns_name" {
  value = "${aws_alb.lb.dns_name}"
}
