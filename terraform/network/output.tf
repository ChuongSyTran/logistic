output "vpc_id" {
  value = aws_vpc.logistic_vpc.id
}

output "subnet_id_1a" {
  value = aws_subnet.logistic_public_subnet_1a.id
}

output "subnet_id_1b" {
  value = aws_subnet.logistic_public_subnet_1b.id
}
