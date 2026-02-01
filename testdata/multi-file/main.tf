# Pattern: Multiple .tf files in same directory
resource "aws_s3_bucket" "main" {
  bucket = "main-bucket"
}
