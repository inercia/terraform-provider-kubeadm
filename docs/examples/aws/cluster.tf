locals {
  tags = "${merge(
    map("Name", var.stack_name,
        "Environment", var.stack_name,
        format("kubernetes.io/cluster/%v", var.stack_name), "owned"),
    var.tags)}"

  # name pattern for the different distros
  ami_name_pattern_map = {
    ubuntu   = "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04*"
    fedora   = ".*Fedora-Cloud-Base.*standard.*"
    opensuse = "openSUSE-Leap-15-*"
  }

  ami_name_pattern = "${lookup(local.ami_name_pattern_map, var.ami_distro)}"

  # owner for the different distros
  ami_owner_map = {
    ubuntu   = "099720109477"
    fedora   = "125523088429"
    opensuse = "679593333241"
  }

  ami_owner = "${lookup(local.ami_owner_map, var.ami_distro)}"

  # ssh user used in the different distros
  ssh_user_map = {
    ubuntu   = "ubuntu"
    fedora   = "fedora"
    opensuse = "ec2-user"
  }

  ssh_user = "${lookup(local.ssh_user_map, var.ami_distro)}"

  ssh_key = "${element(var.authorized_keys, 0)}"
}

provider "aws" {
  region     = "${var.aws_region}"
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"
  profile    = "default"
}

###########################################
# images
###########################################

data "template_file" "cloud-init" {
  template = "${file("cloud-init/cloud-init.yaml.tpl")}"

  vars {
    public_key = "${local.ssh_key}"
  }
}

data "template_cloudinit_config" "cfg" {
  gzip          = false
  base64_encode = false

  part {
    content_type = "text/cloud-config"
    content      = "${data.template_file.cloud-init.rendered}"
  }
}

data "aws_ami" "latest_ami" {
  most_recent = true

  owners = ["${local.ami_owner}"]

  filter {
    name   = "name"
    values = ["${local.ami_name_pattern}"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

###########################################
# networking
###########################################
resource "aws_vpc" "platform" {
  cidr_block           = "${var.vpc_cidr}"
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags                 = "${merge(local.tags, map("Class", "VPC"))}"
}

// list of az which can be access from the current region
data "aws_availability_zones" "az" {
  state = "available"
}

resource "aws_vpc_dhcp_options" "platform" {
  domain_name         = "${var.aws_region}.compute.internal"
  domain_name_servers = ["AmazonProvidedDNS"]
  tags                = "${merge(local.tags, map("Class", "VPCDHCP"))}"
}

resource "aws_vpc_dhcp_options_association" "dns_resolver" {
  dhcp_options_id = "${aws_vpc_dhcp_options.platform.id}"
  vpc_id          = "${aws_vpc.platform.id}"
}

resource "aws_internet_gateway" "platform" {
  tags       = "${merge(local.tags, map("Class", "Gateway"))}"
  vpc_id     = "${aws_vpc.platform.id}"
  depends_on = ["aws_vpc.platform"]
}

resource "aws_subnet" "public" {
  availability_zone       = "${element(data.aws_availability_zones.az.names, 0)}"
  cidr_block              = "${var.public_subnet}"
  depends_on              = ["aws_main_route_table_association.main"]
  map_public_ip_on_launch = true

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-subnet-public-${element(data.aws_availability_zones.az.names, 0)}",
    "Class", "VPC"))}"

  vpc_id = "${aws_vpc.platform.id}"
}

resource "aws_subnet" "private" {
  availability_zone       = "${element(data.aws_availability_zones.az.names, 0)}"
  cidr_block              = "${var.private_subnet}"
  map_public_ip_on_launch = true

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-subnet-private-${element(data.aws_availability_zones.az.names, 0)}",
    "Class", "Subnet"))}"

  vpc_id = "${aws_vpc.platform.id}"
}

resource "aws_route_table" "public" {
  vpc_id = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-route-table-public",
    "Class", "RouteTable"))}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.platform.id}"
  }
}

resource "aws_route_table" "private" {
  vpc_id = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-route-table-private",
    "Class", "RouteTable"))}"

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = "${aws_nat_gateway.nat_gw.id}"
  }
}

resource "aws_main_route_table_association" "main" {
  route_table_id = "${aws_route_table.public.id}"
  vpc_id         = "${aws_vpc.platform.id}"
}

resource "aws_route_table_association" "private" {
  route_table_id = "${aws_route_table.private.id}"
  subnet_id      = "${aws_subnet.private.id}"
}

resource "aws_route_table_association" "public" {
  route_table_id = "${aws_route_table.public.id}"
  subnet_id      = "${aws_subnet.public.id}"
}

resource "aws_eip" "nat_eip" {
  vpc        = true
  depends_on = ["aws_internet_gateway.platform"]
}

resource "aws_nat_gateway" "nat_gw" {
  allocation_id = "${aws_eip.nat_eip.id}"
  subnet_id     = "${aws_subnet.public.id}"
  depends_on    = ["aws_eip.nat_eip"]
}

###########################################
# load balancer
###########################################
resource "aws_elb" "kube_api" {
  connection_draining       = false
  cross_zone_load_balancing = true
  idle_timeout              = 400
  name                      = "${var.stack_name}-elb"
  security_groups           = ["${aws_security_group.elb.id}"]
  subnets                   = ["${aws_subnet.public.0.id}"]
  instances                 = ["${aws_instance.masters.id}"]

  listener {
    instance_port     = 6443
    instance_protocol = "tcp"
    lb_port           = 6443
    lb_protocol       = "tcp"
  }

  listener {
    instance_port     = 6443
    instance_protocol = "tcp"
    lb_port           = 6443
    lb_protocol       = "tcp"
  }

  health_check {
    healthy_threshold   = 2
    interval            = 30
    target              = "TCP:6443"
    timeout             = 3
    unhealthy_threshold = 6
  }
}

output "elb_address" {
  value = "${aws_elb.kube_api.dns_name}"
}

###########################################
# security
###########################################
resource "aws_security_group" "ssh" {
  description = "allow ssh traffic"
  name        = "${var.stack_name}-ssh"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-ssh",
    "Class", "SecurityGroup"))}"

  // allow traffic for TCP 22
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "LBPorts" {
  description = "allow load balancers to hit high ports"
  name        = "${var.stack_name}-lbports"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-lbport",
    "Class", "SecurityGroup"))}"

  // allow traffic for TCP 22
  ingress {
    from_port   = 30000
    to_port     = 32767
    protocol    = "tcp"
    cidr_blocks = ["10.1.0.0/16"]
  }
}

resource "aws_security_group" "icmp" {
  description = "allow ping between instances"
  name        = "${var.stack_name}-icmp"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-icmp",
    "Class", "SecurityGroup"))}"

  ingress {
    from_port       = -1
    to_port         = -1
    protocol        = "icmp"
    security_groups = []
    self            = true
  }

  egress {
    from_port       = -1
    to_port         = -1
    protocol        = "icmp"
    security_groups = []
    cidr_blocks     = ["${var.vpc_cidr}"]
  }
}

resource "aws_security_group" "egress" {
  description = "egress traffic"
  name        = "${var.stack_name}-egress"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-egress",
    "Class", "SecurityGroup"))}"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "kube" {
  description = "give access to everything for now"
  name        = "${var.stack_name}-kube"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-kube",
    "Class", "SecurityGroup"))}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["${var.vpc_cidr}"]
  }

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["${var.private_subnet}"]
  }
}

resource "aws_security_group" "allow_https" {
  description = "give access to 6443 port"
  name        = "${var.stack_name}-allow-https-to-kubeapi"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-https",
    "Class", "SecurityGroup"))}"

  ingress {
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# A security group for the ELB so it is accessible via the web
resource "aws_security_group" "elb" {
  name        = "${var.stack_name}-kube-api"
  description = "give access to kube api server"
  vpc_id      = "${aws_vpc.platform.id}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-elb",
    "Class", "SecurityGroup"))}"

  # HTTP access from anywhere
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # outbound internet access
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_key_pair" "kube" {
  key_name   = "${var.stack_name}-keypair"
  public_key = "${local.ssh_key}"
}

###########################################
# Kubeadm
###########################################

data "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  api {
    # the Load Balancer external, public DNS name
    external = "${aws_elb.kube_api.dns_name}"
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
# bastion host
###########################################
resource "aws_instance" "bastion" {
  ami                         = "${data.aws_ami.latest_ami.id}"
  associate_public_ip_address = true
  count                       = "${var.masters}"
  instance_type               = "${var.master_size}"
  key_name                    = "${aws_key_pair.kube.key_name}"
  source_dest_check           = false
  subnet_id                   = "${aws_subnet.public.0.id}"
  user_data                   = "${data.template_cloudinit_config.cfg.rendered}"

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-master-${count.index}",
    "Class", "Instance"))}"

  vpc_security_group_ids = [
    "${aws_security_group.ssh.id}",
    "${aws_security_group.icmp.id}",
    "${aws_security_group.egress.id}",
    "${aws_security_group.kube.id}",
    "${aws_security_group.allow_https.id}",
  ]

  lifecycle {
    create_before_destroy = true

    # ignore_changes = ["associate_public_ip_address"]
  }

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 20
    delete_on_termination = true
  }
}

###########################################
# master
###########################################
resource "aws_instance" "masters" {
  ami               = "${data.aws_ami.latest_ami.id}"
  count             = "${var.masters}"
  instance_type     = "${var.master_size}"
  key_name          = "${aws_key_pair.kube.key_name}"
  source_dest_check = false
  subnet_id         = "${aws_subnet.private.0.id}"
  user_data         = "${data.template_cloudinit_config.cfg.rendered}"
  depends_on        = ["aws_instance.bastion"]

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-master-${count.index}",
    "Class", "Instance"))}"

  vpc_security_group_ids = [
    "${aws_security_group.ssh.id}",
    "${aws_security_group.icmp.id}",
    "${aws_security_group.egress.id}",
    "${aws_security_group.kube.id}",
    "${aws_security_group.allow_https.id}",
  ]

  lifecycle {
    create_before_destroy = true

    # ignore_changes = ["associate_public_ip_address"]
  }

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 20
    delete_on_termination = true
  }
}

## note: in AWS, the masters must be provisioned in a separate null_resource,
##       as we have the following dependencies:
##
##          aws_instance.masters / provisioner "kubeadm" -> data.kubeadm.main.config
##          data.kubeadm.main -> aws_elb.kube_api.dns_name
##          aws_elb.kube_api.instances -> aws_instance.masters
##
##       so we have a circular dependency: we need the Load Balancer in order to
##       provision instances with kubeadm (as kubeadm tries to access the LB address),
##       and we need the instances in order to add them to the Load Balancer
##
resource "null_resource" "masters" {
  count = "${var.masters}"

  depends_on = [
    "aws_instance.masters",
    "aws_elb.kube_api",
  ]

  connection {
    type         = "ssh"
    user         = "${local.ssh_user}"
    agent        = true
    host         = "${element(aws_instance.masters.*.private_ip, count.index)}"
    bastion_host = "${aws_instance.bastion.public_ip}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot prooperly detect a valid hostname
    #nodename = "${aws_instance.masters.private_dns}"
    nodename = "${element(aws_instance.masters.*.private_dns, count.index)}"

    install {
      auto = true
    }
  }
}

output "master.public_ip" {
  value = "${aws_instance.masters.*.private_ip}"
}

output "master.private_dns" {
  value = "${aws_instance.masters.*.private_dns}"
}

###########################################
# workers
###########################################

resource "aws_instance" "workers" {
  ami               = "${data.aws_ami.latest_ami.id}"
  count             = "${var.workers}"
  instance_type     = "${var.worker_size}"
  key_name          = "${aws_key_pair.kube.key_name}"
  source_dest_check = false
  subnet_id         = "${aws_subnet.private.0.id}"
  user_data         = "${data.template_cloudinit_config.cfg.rendered}"
  depends_on        = ["null_resource.masters"]

  tags = "${merge(local.tags, map(
    "Name", "${var.stack_name}-node-${count.index}",
    "Class", "Instance"))}"

  security_groups = [
    "${aws_security_group.ssh.id}",
    "${aws_security_group.icmp.id}",
    "${aws_security_group.egress.id}",
    "${aws_security_group.kube.id}",
    "${aws_security_group.LBPorts.id}",
  ]

  lifecycle {
    create_before_destroy = true

    # ignore_changes = ["associate_public_ip_address"]
  }

  root_block_device {
    volume_type           = "gp2"
    volume_size           = 20
    delete_on_termination = true
  }

  connection {
    type         = "ssh"
    user         = "${local.ssh_user}"
    agent        = true
    host         = "${self.private_ip}"
    bastion_host = "${aws_instance.bastion.public_ip}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"
    join   = "${element(aws_instance.masters.*.private_ip, 0)}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot properly detect a valid hostname
    nodename = "${self.private_dns}"

    install {
      auto = true
    }
  }
}

output "workers.public_ip" {
  value = "${aws_instance.workers.*.public_ip}"
}

output "workers.private_dns" {
  value = "${aws_instance.workers.*.private_dns}"
}
