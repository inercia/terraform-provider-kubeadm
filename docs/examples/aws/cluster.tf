locals {
  tags = "${merge(
    map("Name", var.stack_name,
        "Environment", var.stack_name,
        format("kubernetes.io/cluster/%v", var.stack_name), "owned"),
    var.tags)}"

  # name pattern for the different distros
  ami_name_pattern_map = {
    ubuntu = "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04*"
    fedora = ".*Fedora-Cloud-Base.*standard.*"
  }

  ami_name_pattern = "${lookup(local.ami_name_pattern_map, var.ami_distro)}"

  # owner for the different distros
  ami_owner_map = {
    ubuntu = "099720109477"
    fedora = "125523088429"
  }

  ami_owner = "${lookup(local.ami_owner_map, var.ami_distro)}"

  # ssh user used in the different distros
  ssh_user_map = {
    ubuntu = "ubuntu"
    fedora = "fedora"
  }

  ssh_user = "${lookup(local.ssh_user_map, var.ami_distro)}"
}

###########################################
#
###########################################

provider "aws" {
  region     = "${var.aws_region}"
  access_key = "${var.aws_access_key}"
  secret_key = "${var.aws_secret_key}"

  #shared_credentials_file = "~/.aws/creds"
  profile = "default"
}

# resource "aws_iam_role" "k8s_master" {
#   assume_role_policy = <<EOF
# {
#   "Version": "2012-10-17",
#   "Statement": [
#     {
#       "Effect": "Allow",
#       "Principal": { "Service": "ec2.amazonaws.com" },
#       "Action": "sts:AssumeRole"
#     }
#   ]
# }
# EOF
# }
#
# resource "aws_iam_role" "k8s_worker" {
#   assume_role_policy = <<EOF
# {
#   "Version": "2012-10-17",
#   "Statement": [
#     {
#       "Effect": "Allow",
#       "Principal": { "Service": "ec2.amazonaws.com"},
#       "Action": "sts:AssumeRole"
#     }
#   ]
# }
# EOF
# }
#
# resource "aws_iam_role_policy" "k8s_master" {
#   role = "${aws_iam_role.k8s_master.id}"
#   policy = <<EOF
# {
#   "Version": "2012-10-17",
#   "Statement": [
#     {
#       "Effect": "Allow",
#       "Action": ["ec2:*"],
#       "Resource": ["*"]
#     },
#     {
#       "Effect": "Allow",
#       "Action": ["elasticloadbalancing:*"],
#       "Resource": ["*"]
#     },
#     {
#       "Effect": "Allow",
#       "Action": ["route53:*"],
#       "Resource": ["*"]
#     },
#     {
#       "Effect": "Allow",
#       "Action": "s3:*",
#       "Resource": [ "arn:aws:s3:::kubernetes-*"]
#     }
#   ]
# }
# EOF
# }
#
# resource "aws_iam_role_policy" "k8s_worker" {
#   role = "${aws_iam_role.k8s_worker.id}"
#   policy = <<EOF
# {
#   "Version": "2012-10-17",
#   "Statement": [
#     {
#       "Effect": "Allow",
#       "Action": "s3:*",
#       "Resource": [
#         "arn:aws:s3:::kubernetes-*"
#       ]
#     },
#     {
#       "Effect": "Allow",
#       "Action": "ec2:Describe*",
#       "Resource": "*"
#     },
#     {
#       "Effect": "Allow",
#       "Action": "ec2:AttachVolume",
#       "Resource": "*"
#     },
#     {
#       "Effect": "Allow",
#       "Action": "ec2:DetachVolume",
#       "Resource": "*"
#     },
#     {
#       "Effect": "Allow",
#       "Action": ["route53:*"],
#       "Resource": ["*"]
#     },
#     {
#       "Effect": "Allow",
#       "Action": [
#         "ecr:GetAuthorizationToken",
#         "ecr:BatchCheckLayerAvailability",
#         "ecr:GetDownloadUrlForLayer",
#         "ecr:GetRepositoryPolicy",
#         "ecr:DescribeRepositories",
#         "ecr:ListImages",
#         "ecr:BatchGetImage"
#       ],
#       "Resource": "*"
#     }
#   ]
# }
# EOF
# }
#
# resource "aws_iam_instance_profile" "k8s_master" {
#   name = "profile_master"
#   role = "${aws_iam_role.k8s_master.name}"
# }
#
# resource "aws_iam_instance_profile" "k8s_worker" {
#   name = "profile_worker"
#   role = "${aws_iam_role.k8s_worker.name}"
# }

resource "aws_vpc" "vpc" {
  cidr_block           = "${var.vpc_cidr}"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = "${merge(local.tags, map("Class", "VPC"))}"
}

resource "aws_internet_gateway" "igw" {
  vpc_id     = "${aws_vpc.vpc.id}"
  depends_on = ["aws_vpc.vpc"]
}

resource "aws_subnet" "public_subnet" {
  vpc_id                  = "${aws_vpc.vpc.id}"
  cidr_block              = "${var.vpc_cidr_public}"
  availability_zone       = "${var.aws_az}"
  map_public_ip_on_launch = true
  depends_on              = ["aws_vpc.vpc"]
  tags                    = "${merge(local.tags, map("Class", "PublicSubnet"))}"
}

resource "aws_subnet" "private_subnet" {
  vpc_id            = "${aws_vpc.vpc.id}"
  cidr_block        = "${var.vpc_cidr_private}"
  availability_zone = "${var.aws_az}"
  depends_on        = ["aws_vpc.vpc"]
  tags              = "${merge(local.tags, map("Class", "PrivateSubnet"))}"
}

resource "aws_eip" "nat_eip" {
  vpc        = true
  depends_on = ["aws_internet_gateway.igw"]
}

resource "aws_nat_gateway" "nat_gw" {
  allocation_id = "${aws_eip.nat_eip.id}"
  subnet_id     = "${aws_subnet.public_subnet.id}"
  depends_on    = ["aws_eip.nat_eip"]
}

resource "aws_route_table" "public_rt" {
  vpc_id = "${aws_vpc.vpc.id}"

  route {
    gateway_id = "${aws_internet_gateway.igw.id}"
    cidr_block = "0.0.0.0/0"
  }

  depends_on = ["aws_vpc.vpc"]
}

resource "aws_route_table" "private_rt" {
  vpc_id     = "${aws_vpc.vpc.id}"
  depends_on = ["aws_vpc.vpc"]
  tags       = "${merge(local.tags, map("Class", "PrivateRouteTable"))}"

  route = {
    nat_gateway_id = "${aws_nat_gateway.nat_gw.id}"
    cidr_block     = "0.0.0.0/0"
  }
}

resource "aws_route_table_association" "public_rta" {
  subnet_id      = "${aws_subnet.public_subnet.id}"
  route_table_id = "${aws_route_table.public_rt.id}"
  depends_on     = ["aws_subnet.public_subnet"]
}

resource "aws_route_table_association" "private_rta" {
  subnet_id      = "${aws_subnet.private_subnet.id}"
  route_table_id = "${aws_route_table.private_rt.id}"
  depends_on     = ["aws_subnet.private_subnet"]
}

resource "aws_elb" "k8s_master_elb" {
  name                      = "k8scontrollerselb"
  subnets                   = ["${aws_subnet.public_subnet.id}"]
  security_groups           = ["${aws_security_group.allow_k8s_elb.id}"]
  instances                 = ["${aws_instance.k8s_master.*.id}"]
  cross_zone_load_balancing = false

  listener {
    instance_port     = 80
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }

  listener {
    lb_port           = 6443
    lb_protocol       = "tcp"
    instance_port     = 6443
    instance_protocol = "tcp"
  }

  # we must use really flexible timeouts in order to give
  # some time until the API server is up and running...
  health_check {
    healthy_threshold   = 2
    unhealthy_threshold = 10
    timeout             = 10
    target              = "TCP:6443"
    interval            = 25
  }
  depends_on = ["aws_instance.k8s_master"]
  tags       = "${merge(local.tags, map("Class", "PrivateSubnet"))}"
}

resource "aws_security_group" "allow_k8s_elb" {
  name        = "allow_k8s_elb"
  description = "Allow trafic to K8S controllers"
  vpc_id      = "${aws_vpc.vpc.id}"

  ingress {
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "allow_ssh_to_public" {
  name        = "allow_ssh_public"
  description = "Allow SSH from internet to the public subnet"
  vpc_id      = "${aws_vpc.vpc.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "allow_any_public_to_private" {
  name        = "allow_ssh_public_to_private_net"
  description = "Allow any traffic from public subnet and elb to the private subnet"
  vpc_id      = "${aws_vpc.vpc.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["${aws_subnet.public_subnet.cidr_block}"]
  }

  ingress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    security_groups = ["${aws_security_group.allow_k8s_elb.id}"]
  }

  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

data "aws_ami" "image" {
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

resource "aws_key_pair" "kp" {
  key_name   = "${var.stack_name}"
  public_key = "${file("${var.private_key}.pub")}"
}

data "template_file" "cloud-init" {
  template = "${file("cloud-init/cloud-init.yaml.tpl")}"

  vars {
    public_key = "${file("${var.private_key}.pub")}"
  }
}

resource "aws_instance" "bastion" {
  ami                         = "${data.aws_ami.image.id}"
  instance_type               = "t2.micro"
  availability_zone           = "${var.aws_az}"
  subnet_id                   = "${aws_subnet.public_subnet.id}"
  private_ip                  = "${cidrhost(aws_subnet.public_subnet.cidr_block, 10)}"
  vpc_security_group_ids      = ["${aws_security_group.allow_ssh_to_public.id}"]
  key_name                    = "${aws_key_pair.kp.key_name}"
  associate_public_ip_address = true
  user_data                   = "${data.template_file.cloud-init.rendered}"
  tags                        = "${merge(local.tags, map("Name", format("%s-bastion", var.stack_name), "Class", "Instance"))}"
}

resource "aws_instance" "k8s_master" {
  count                       = "${var.masters}"
  ami                         = "${data.aws_ami.image.id}"
  instance_type               = "${var.master_size}"
  availability_zone           = "${var.aws_az}"
  subnet_id                   = "${aws_subnet.private_subnet.id}"
  private_ip                  = "${cidrhost(aws_subnet.private_subnet.cidr_block, 10 + count.index)}"
  vpc_security_group_ids      = ["${aws_security_group.allow_any_public_to_private.id}"]
  key_name                    = "${aws_key_pair.kp.key_name}"
  associate_public_ip_address = false
  user_data                   = "${data.template_file.cloud-init.rendered}"

  # iam_instance_profile        = "${aws_iam_instance_profile.k8s_master.name}"
  depends_on = ["aws_nat_gateway.nat_gw"]
  tags       = "${merge(local.tags, map("Name", format("%s-master-%d", var.stack_name, count.index), "Class", "Instance"))}"
}

resource "aws_instance" "k8s_worker" {
  count                       = "${var.workers}"
  ami                         = "${data.aws_ami.image.id}"
  instance_type               = "${var.worker_size}"
  availability_zone           = "${var.aws_az}"
  subnet_id                   = "${aws_subnet.private_subnet.id}"
  private_ip                  = "${cidrhost(aws_subnet.private_subnet.cidr_block, 30 + count.index)}"
  vpc_security_group_ids      = ["${aws_security_group.allow_any_public_to_private.id}"]
  key_name                    = "${aws_key_pair.kp.key_name}"
  associate_public_ip_address = false
  user_data                   = "${data.template_file.cloud-init.rendered}"

  # iam_instance_profile        = "${aws_iam_instance_profile.k8s_worker.name}"
  depends_on = ["aws_nat_gateway.nat_gw"]
  tags       = "${merge(local.tags, map("Name", format("%s-worker-%d", var.stack_name, count.index), "Class", "Instance"))}"
}

###########################################
# Kubeadm
###########################################

# we must do all the kubeadm stuff _after_ we have created the Load Balancer
# and the Load Balancer is going to be created _after_ the masters.
# so our `kubeadm` data/provisioners must be done at the end,
# in some `null_resources`

data "kubeadm" "main" {
  config_path = "${var.kubeconfig}"

  api {
    external = "${aws_elb.k8s_master_elb.dns_name}"
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

resource "null_resource" "master" {
  count      = "${var.masters}"
  depends_on = ["aws_instance.k8s_master"]

  connection {
    type         = "ssh"
    user         = "${local.ssh_user}"
    private_key  = "${file(var.private_key)}"
    host         = "${element(aws_instance.k8s_master.*.private_ip, count.index)}"
    bastion_host = "${aws_instance.bastion.public_ip}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot prooperly detect a valid hostname
    nodename = "${aws_instance.k8s_master.private_dns}"

    install {
      auto = true
    }
  }
}

resource "null_resource" "worker" {
  count      = "${var.workers}"
  depends_on = ["aws_instance.k8s_worker", "null_resource.master"]

  connection {
    type         = "ssh"
    user         = "${local.ssh_user}"
    private_key  = "${file(var.private_key)}"
    host         = "${element(aws_instance.k8s_worker.*.private_ip, count.index)}"
    bastion_host = "${aws_instance.bastion.public_ip}"
  }

  provisioner "kubeadm" {
    config = "${data.kubeadm.main.config}"
    join   = "${element(aws_instance.k8s_master.*.private_ip, 0)}"

    # we must overwrite the nodename with the AWS private DNS name
    # because the kubelet cannot prooperly detect a valid hostname
    nodename = "${aws_instance.k8s_worker.private_dns}"

    install {
      auto = true
    }
  }
}

###########################################
# output
###########################################

output "ip_bastion" {
  value = "${aws_instance.bastion.public_ip}"
}

output "dns_lb" {
  value = "${aws_elb.k8s_master_elb.dns_name}"
}

output "ip_masters" {
  value = [
    "${aws_instance.k8s_master.*.private_ip}",
  ]
}

output "ip_workers" {
  value = [
    "${aws_instance.k8s_worker.*.private_ip}",
  ]
}
