locals {
  base_instance_type = "c5.xlarge"
  nsqd_instance_type = "c5.2xlarge"
  key_name           = "ck-mac"
}

module "monitoring" {
  source         = "../../modules/ec2"
  instance_count = 1

  role = "monitoring"

  instance_type          = local.base_instance_type
  ami                    = data.aws_ami.amazon-linux-2.id
  key_name               = local.key_name
  vpc_security_group_ids = [module.vpc.default_security_group_id, aws_security_group.allow_ssh.id, aws_security_group.allow_internal_all.id]
  subnet_id              = module.vpc.public_subnets[0]
}

locals {
    secondary_range = cidrsubnet(module.vpc.public_subnets_cidr_blocks[0], 2, 1)
}

resource "aws_ec2_subnet_cidr_reservation" "reserved" {
  cidr_block       = local.secondary_range
  reservation_type = "explicit"
  subnet_id        = module.vpc.public_subnets[0]
}

module "consumer" {
  source         = "../../modules/ec2"
  instance_count = 3

  role = "consumer"

  instance_type          = local.nsqd_instance_type
  ami                    = data.aws_ami.amazon-linux-2.id
  key_name               = local.key_name
  vpc_security_group_ids = [module.vpc.default_security_group_id, aws_security_group.allow_ssh.id, aws_security_group.allow_internal_all.id]
  subnet_id              = module.vpc.public_subnets[0]
  secondary_private_ip_range = local.secondary_range
  secondary_private_ip_space = 3
}

module "nsqd" {
  source         = "../../modules/ec2"
  instance_count = 3

  role = "nsqd"

  instance_type          = local.nsqd_instance_type
  ami                    = data.aws_ami.amazon-linux-2.id
  key_name               = local.key_name
  vpc_security_group_ids = [module.vpc.default_security_group_id, aws_security_group.allow_ssh.id, aws_security_group.allow_internal_all.id]
  subnet_id              = module.vpc.public_subnets[0]
}

module "nsqlookupd" {
  source         = "../../modules/ec2"
  instance_count = 1

  role = "nsqlookupd"

  instance_type          = local.base_instance_type
  ami                    = data.aws_ami.amazon-linux-2.id
  key_name               = local.key_name
  vpc_security_group_ids = [module.vpc.default_security_group_id, aws_security_group.allow_ssh.id, aws_security_group.allow_internal_all.id]
  subnet_id              = module.vpc.public_subnets[0]
}

resource "aws_security_group" "allow_ssh" {
  name        = "allow_ssh"
  description = "Allow ssh traffic"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port        = 22
    to_port          = 22
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  tags = {
    Name = "allow_ssh"
  }
}

resource "aws_security_group" "allow_internal_all" {
  name        = "allow_internal_all"
  description = "Allow ssh traffic"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = [module.vpc.vpc_cidr_block]
  }

  tags = {
    Name = "allow_internal_all"
  }
}

data "aws_ami" "amazon-linux-2" {
  most_recent = true

  filter {
    name   = "owner-alias"
    values = ["amazon"]
  }

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-ebs"]
  }
}

