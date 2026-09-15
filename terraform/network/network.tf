# Tạo VPC 65,536 địa chỉ IP
resource "aws_vpc" "logistic_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "logistic-production-vpc"
  }
}

resource "aws_subnet" "logistic_public_subnet_1a" {
  vpc_id                  = aws_vpc.logistic_vpc.id # Khai kháo subnet này thuộc vpc nào
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "ap-southeast-1a" # Đặt cố định ở Data Center 1a

  tags = {
    Name = "logistic-public-subnet-1a"
  }
}

resource "aws_subnet" "logistic_public_subnet_1b" {
  vpc_id                  = aws_vpc.logistic_vpc.id # Khai kháo subnet này thuộc vpc nào
  cidr_block              = "10.0.2.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "ap-southeast-1b" # Đặt cố định ở Data Center 1b

  tags = {
    Name = "logistic-public-subnet-1b"
  }
}

resource "aws_eip" "logistic_public_ip_1a" {
  domain                    = "vpc"
}

resource "aws_eip" "logistic_public_ip_1b" {
  domain = "vpc"
}

resource "aws_internet_gateway" "logistic_igw" {
  vpc_id = aws_vpc.logistic_vpc.id

  tags = {
    Name = "logistic-igw"
  }
}

resource "aws_nat_gateway" "logistic_ngw_1a" {
  allocation_id = aws_eip.logistic_public_ip_1a.id
  subnet_id = aws_subnet.logistic_public_subnet_1a.id
  availability_mode = "zonal"

  tags = {
    Name = "logistic-ngw-1a"
  }

  depends_on = [ aws_internet_gateway.logistic_igw ]
}

resource "aws_nat_gateway" "logistic_ngw_1b" {
  allocation_id = aws_eip.logistic_public_ip_1b.id
  subnet_id = aws_subnet.logistic_public_subnet_1b.id
  availability_mode = "zonal"

  tags = {
    Name = "logistic-ngw-1b"
  }

  depends_on = [ aws_internet_gateway.logistic_igw ]
}

resource "aws_subnet" "logistic_private_subnet_1a" {
  vpc_id                  = aws_vpc.logistic_vpc.id
  cidr_block              = "10.0.3.0/24"
  map_public_ip_on_launch = false
  availability_zone       = "ap-southeast-1a"

  tags = {
    Name = "logistic-private-subnet-1a"
  }
}

resource "aws_subnet" "logistic_private_subnet_1b" {
  vpc_id                  = aws_vpc.logistic_vpc.id
  cidr_block              = "10.0.4.0/24"
  map_public_ip_on_launch = false
  availability_zone       = "ap-southeast-1b"

  tags = {
    Name = "logistic-private-subnet-1b"
  }
}

resource "aws_route_table" "logistic_rt_public" {
  vpc_id = aws_vpc.logistic_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.logistic_igw.id
  }

  tags = {
    Name = "logistic-public-rt"
  }
}

resource "aws_route_table" "logistic_rt_private_1a" {
  vpc_id = aws_vpc.logistic_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.logistic_ngw_1a.id
  }

  tags = {
    Name = "logistic-rt-private-1a"
  }
}

resource "aws_route_table" "logistic_rt_private_1b" {
  vpc_id = aws_vpc.logistic_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.logistic_ngw_1b.id
  }

  tags = {
    Name = "logistic-rt-private-1b"
  }
}

resource "aws_route_table_association" "logistic_public_1a_assoc" {
  subnet_id      = aws_subnet.logistic_public_subnet_1a.id
  route_table_id = aws_route_table.logistic_rt_public.id
}

resource "aws_route_table_association" "logistic_public_1b_assoc" {
  subnet_id      = aws_subnet.logistic_public_subnet_1b.id
  route_table_id = aws_route_table.logistic_rt_public.id
}

resource "aws_route_table_association" "logistic_private_1a_assoc" {
  subnet_id      = aws_subnet.logistic_private_subnet_1a.id
  route_table_id = aws_route_table.logistic_rt_private_1a.id
}

resource "aws_route_table_association" "logistic_private_1b_assoc" {
  subnet_id      = aws_subnet.logistic_private_subnet_1b.id
  route_table_id = aws_route_table.logistic_rt_private_1b.id
}
