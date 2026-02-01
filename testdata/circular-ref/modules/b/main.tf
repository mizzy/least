# Module B - calls module A (circular reference)
resource "aws_sqs_queue" "from_b" {
  name = "queue-from-b"
}

# This creates a circular reference: main -> a -> b -> a
module "a" {
  source = "../a"
}
