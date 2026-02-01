# Pattern: Resources with for_each and count
variable "buckets" {
  default = {
    logs   = { versioning = true }
    data   = { versioning = true }
    backup = { versioning = false }
  }
}

variable "queue_count" {
  default = 3
}

# for_each with map
resource "aws_s3_bucket" "multi" {
  for_each = var.buckets
  bucket   = "${each.key}-bucket"
}

resource "aws_s3_bucket_versioning" "multi" {
  for_each = var.buckets
  bucket   = aws_s3_bucket.multi[each.key].id

  versioning_configuration {
    status = each.value.versioning ? "Enabled" : "Disabled"
  }
}

# count
resource "aws_sqs_queue" "workers" {
  count = var.queue_count
  name  = "worker-queue-${count.index}"
}

# for_each with toset
resource "aws_sns_topic" "alerts" {
  for_each = toset(["critical", "warning", "info"])
  name     = "${each.key}-alerts"
}

# Dynamic blocks
resource "aws_security_group" "main" {
  name   = "main-sg"
  vpc_id = "vpc-12345"

  dynamic "ingress" {
    for_each = [80, 443, 8080]
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
}
