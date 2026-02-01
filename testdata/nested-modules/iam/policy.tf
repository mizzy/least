data "aws_iam_policy_document" "least_privilege" {
  statement {
    sid    = "LeastPrivilege"
    effect = "Allow"

    actions = [
      "ecs:CreateCluster",
      "ecs:CreateService",
      "ecs:DeleteCluster",
      "ecs:DeleteService",
      "ecs:DeregisterTaskDefinition",
      "ecs:DescribeClusters",
      "ecs:DescribeServices",
      "ecs:DescribeTaskDefinition",
      "ecs:ListTagsForResource",
      "ecs:RegisterTaskDefinition",
      "ecs:TagResource",
      "ecs:UntagResource",
      "ecs:UpdateCluster",
      "ecs:UpdateService",
      "iam:PassRole",
      "rds:AddTagsToResource",
      "rds:CreateDBCluster",
      "rds:DeleteDBCluster",
      "rds:DescribeDBClusters",
      "rds:ListTagsForResource",
      "rds:ModifyDBCluster",
      "rds:RemoveTagsFromResource",
    ]

    resources = [
      "*",
    ]
  }
}

