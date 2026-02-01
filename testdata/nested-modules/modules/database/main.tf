variable "db_name" {
  type = string
}

resource "aws_rds_cluster" "this" {
  cluster_identifier = var.db_name
  engine             = "aurora-postgresql"
  engine_mode        = "provisioned"
  database_name      = "mydb"
  master_username    = "admin"
  master_password    = "changeme123"

  serverlessv2_scaling_configuration {
    max_capacity = 1.0
    min_capacity = 0.5
  }
}

resource "aws_rds_cluster_instance" "this" {
  cluster_identifier = aws_rds_cluster.this.id
  instance_class     = "db.serverless"
  engine             = aws_rds_cluster.this.engine
  engine_version     = aws_rds_cluster.this.engine_version
}
