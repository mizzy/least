# Database resources in separate file
resource "aws_dynamodb_table" "users" {
  name         = "users"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }
}

resource "aws_rds_cluster" "main" {
  cluster_identifier = "main-cluster"
  engine             = "aurora-mysql"
  master_username    = "admin"
  master_password    = "changeme"
}
