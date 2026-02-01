# IAM resources with policy documents
resource "aws_iam_role" "app" {
  name = "app-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

data "aws_iam_policy_document" "app_policy" {
  statement {
    sid    = "S3Access"
    effect = "Allow"

    actions = [
      "s3:GetObject",
      "s3:PutObject",
    ]

    resources = [
      "arn:aws:s3:::my-bucket/*",
    ]
  }

  statement {
    sid    = "DynamoDBAccess"
    effect = "Allow"

    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:Query",
    ]

    resources = [
      "arn:aws:dynamodb:*:*:table/users",
    ]
  }
}

resource "aws_iam_role_policy" "app" {
  name   = "app-policy"
  role   = aws_iam_role.app.id
  policy = data.aws_iam_policy_document.app_policy.json
}
