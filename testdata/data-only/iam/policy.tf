data "aws_iam_policy_document" "least_privilege" {
  statement {
    sid    = "LeastPrivilege"
    effect = "Allow"

    actions = [
    ]

    resources = [
      "*",
    ]
  }
}

