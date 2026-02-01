# Pattern: Relative path with ../
# Run from: testdata/relative-path/envs/prod

resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
}

module "shared_vpc" {
  source = "../../shared"
}
