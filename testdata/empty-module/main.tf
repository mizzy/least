# Pattern: Module that contains no resources
resource "aws_s3_bucket" "main" {
  bucket = "main-bucket"
}

module "placeholder" {
  source = "./modules/placeholder"
}
