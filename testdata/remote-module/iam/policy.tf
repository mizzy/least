data "aws_iam_policy_document" "least_privilege" {
  statement {
    sid    = "LeastPrivilege"
    effect = "Allow"

    actions = [
      "ec2:CreateTags",
      "ec2:CreateVpc",
      "ec2:DeleteTags",
      "ec2:DeleteVpc",
      "ec2:DescribeVpcAttribute",
      "ec2:DescribeVpcs",
      "ec2:ModifyVpcAttribute",
    ]

    resources = [
      "*",
    ]
  }
}
