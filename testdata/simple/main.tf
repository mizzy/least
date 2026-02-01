# Pattern: Simple - No modules, just resources
resource "aws_s3_bucket" "main" {
  bucket = "my-bucket"
}

resource "aws_dynamodb_table" "main" {
  name         = "my-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
}
