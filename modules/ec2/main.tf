
resource "aws_instance" "host" {
  count = var.instance_count

  instance_type          = var.instance_type
  key_name               = var.key_name
  vpc_security_group_ids = var.vpc_security_group_ids
  subnet_id              = var.subnet_id

  ami                         = var.ami
  associate_public_ip_address = true

  secondary_private_ips = var.secondary_private_ip_range != "" ? [for i in range(pow(2, var.secondary_private_ip_space) - 1): cidrhost(cidrsubnet(var.secondary_private_ip_range, var.secondary_private_ip_space, count.index + 1), i)] : []

  tags = {
    Name = "${var.role}-${count.index}"
  }

}


resource "local_file" "inventory" {
  filename = "./ansible/inventory/${var.role}.toml"
  content = templatefile("${path.module}/inventory.tpl", {
    role     = var.role,
    hosts    = aws_instance.host
    ssh_user = var.ssh_user
  })
}
