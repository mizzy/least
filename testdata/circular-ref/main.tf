# Pattern: Circular reference - A calls B, B calls A
# Tests that visited tracking prevents infinite loops
resource "aws_s3_bucket" "root" {
  bucket = "root-bucket"
}

module "a" {
  source = "./modules/a"
}
