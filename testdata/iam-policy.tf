# IAM policy using aws_iam_policy_document data source
data "aws_iam_policy_document" "terraform_policy" {
  statement {
    sid    = "S3Access"
    effect = "Allow"

    actions = [
      "s3:CreateBucket",
      "s3:DeleteBucket",
      "s3:GetBucket*",
      "s3:ListBucket",
      "s3:PutBucket*",
    ]

    resources = ["*"]
  }

  statement {
    sid    = "EC2Access"
    effect = "Allow"

    actions = [
      "ec2:DescribeInstances",
      "ec2:RunInstances",
      "ec2:TerminateInstances",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_policy" "terraform_policy" {
  name        = "terraform-policy"
  description = "Policy for Terraform operations"
  policy      = data.aws_iam_policy_document.terraform_policy.json
}
