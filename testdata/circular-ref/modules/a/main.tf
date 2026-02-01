# Module A - calls module B
resource "aws_dynamodb_table" "from_a" {
  name         = "table-from-a"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
}

module "b" {
  source = "../b"
}
