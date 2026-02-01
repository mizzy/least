# Pattern: Local module with ./
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

module "s3" {
  source      = "./modules/s3"
  bucket_name = "my-bucket"
}

module "lambda" {
  source        = "./modules/lambda"
  function_name = "my-function"
}
