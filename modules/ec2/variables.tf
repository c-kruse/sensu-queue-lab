variable "instance_count" {}

variable "role" {}

variable "instance_type" {}
variable "ami" {}
variable "key_name" {}
variable "vpc_security_group_ids" {}
variable "subnet_id" {}
variable "ssh_user" { default = "ec2-user" }
variable "secondary_private_ip_space" {
    description = "allocates 2^secondary_private_ip_space private ips"
    type = number
    default = 0
}
variable "secondary_private_ip_range" {
    description = "cidr block to allocate secondary ips from"
    type = string
    default = ""
}
