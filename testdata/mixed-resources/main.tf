# Pattern: Mixed resources from multiple AWS services
resource "aws_s3_bucket" "data" {
  bucket = "data-bucket"
}

resource "aws_dynamodb_table" "users" {
  name         = "users"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "user_id"

  attribute {
    name = "user_id"
    type = "S"
  }
}

resource "aws_sqs_queue" "events" {
  name = "events-queue"
}

resource "aws_sns_topic" "alerts" {
  name = "alerts-topic"
}

resource "aws_lambda_function" "processor" {
  function_name = "event-processor"
  role          = aws_iam_role.lambda.arn
  handler       = "index.handler"
  runtime       = "python3.11"
  filename      = "lambda.zip"
}

resource "aws_iam_role" "lambda" {
  name = "lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_secretsmanager_secret" "api_key" {
  name = "api-key"
}

resource "aws_kms_key" "main" {
  description = "Main encryption key"
}
