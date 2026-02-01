# Shared module accessed via ../
resource "aws_vpc" "shared" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = "shared-vpc"
  }
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.shared.id
  cidr_block = "10.0.1.0/24"

  tags = {
    Name = "public-subnet"
  }
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.shared.id
}
