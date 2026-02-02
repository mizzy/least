# least

> [!CAUTION]
> This is an experimental project and has not been thoroughly tested. It may not work as expected.
> This project is subject to breaking changes without notice.

Generate least-privilege IAM policies from Infrastructure-as-Code.

`least` analyzes your Terraform configurations and generates minimal IAM policies required to manage the defined resources. It can also check existing policies against requirements to identify over/under permissions.

## Features

- **Generate**: Create minimal IAM policies from Terraform code
- **Specific ARNs**: Generate resource-specific ARNs instead of wildcards for true least-privilege
- **Check**: Compare existing policies against requirements (for CI/CD)
- **Multi-provider**: Extensible architecture for future IaC tool support

## Installation

```bash
go install github.com/mizzy/least/cmd/least@latest
```

Or build from source:

```bash
git clone https://github.com/mizzy/least.git
cd least
go build -o least ./cmd/least
```

## Usage

### Generate IAM Policy

```bash
# Analyze Terraform files and output as Terraform HCL (default)
least generate ./terraform

# Output as JSON
least generate ./terraform -f json

# Save to file
least generate ./terraform -o policy.tf
```

Example output (default: Terraform HCL):

```hcl
data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_iam_policy_document" "least_privilege" {
  statement {
    sid    = "AwsS3BucketMain"
    effect = "Allow"

    actions = [
      "s3:CreateBucket",
      "s3:DeleteBucket",
      # ... other S3 actions
    ]

    resources = [
      "arn:aws:s3:::my-bucket",
      "arn:aws:s3:::my-bucket/*",
    ]
  }
  statement {
    sid    = "AwsDynamodbTableMain"
    effect = "Allow"

    actions = [
      "dynamodb:CreateTable",
      "dynamodb:DeleteTable",
      "dynamodb:DescribeTable",
      # ... other DynamoDB actions
    ]

    resources = [
      "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/my-table",
    ]
  }
}
```

Example output (JSON format with `-f json`):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AwsS3BucketMain",
      "Effect": "Allow",
      "Action": [
        "s3:CreateBucket",
        "s3:DeleteBucket"
      ],
      "Resource": [
        "arn:aws:s3:::my-bucket",
        "arn:aws:s3:::my-bucket/*"
      ]
    },
    {
      "Sid": "AwsDynamodbTableMain",
      "Effect": "Allow",
      "Action": [
        "dynamodb:CreateTable",
        "dynamodb:DeleteTable",
        "dynamodb:DescribeTable"
      ],
      "Resource": [
        "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/my-table"
      ]
    }
  ]
}
```

### Check Policy Compliance

Compare an existing IAM policy against requirements:

```bash
# Check against a JSON policy file
least check ./terraform -p existing-policy.json

# Check against Terraform-defined IAM policies
least check ./terraform -d ./iam-policies
```

Exit codes:
- `0`: Compliant
- `1`: Missing permissions (required but not granted)
- `2`: Excessive permissions only (granted but not required)

Example output:

```
Found 5 resources in: ./terraform
Loading IAM policy from JSON: existing-policy.json
✗ Missing permissions (required but not granted):
  - ec2:CreateSecurityGroup
  - ec2:DeleteSecurityGroup
⚠ Excessive permissions (granted but not required):
  + s3:*
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Check IAM Policy
  run: |
    least check ./terraform -p policy.json
```

## How It Works

`least` uses CloudFormation Resource Schemas as the authoritative source for IAM permissions. Each AWS resource type has a schema that defines the exact IAM actions required for create, read, update, and delete operations.

```
Terraform Resource          CloudFormation Schema         IAM Actions
aws_s3_bucket        →     AWS::S3::Bucket        →     s3:CreateBucket, ...
aws_lambda_function  →     AWS::Lambda::Function  →     lambda:CreateFunction, ...
```

### Resource-Specific ARNs

`least` generates specific ARNs for each resource instead of wildcards:

- Extracts resource identifiers (bucket names, table names, etc.) from Terraform configs
- Uses `data.aws_caller_identity.current.account_id` and `data.aws_region.current.name` for dynamic values
- Generates per-resource policy statements with descriptive Sid names
- Falls back to wildcards only for resources with runtime-generated IDs (e.g., EC2 instances)

## Supported Resources

Currently supports 40+ common AWS resource types including:

- **Compute**: EC2, Lambda, ECS, EKS
- **Storage**: S3, DynamoDB, RDS
- **Networking**: VPC, Subnet, Security Group, ALB
- **IAM**: Role, Policy, User, Group
- **Others**: SNS, SQS, KMS, CloudWatch, Route53, and more

See [internal/mapping/mapping.go](internal/mapping/mapping.go) for the full list.

## Development

### Updating Permission Mappings

Permission mappings are generated from CloudFormation schemas:

```bash
# 1. Fetch schemas from AWS (requires AWS CLI configured)
./scripts/fetch-schemas.sh

# 2. Generate Go code from schemas
go generate ./internal/mapping

# 3. Commit the updated generated.go
git add internal/mapping/generated.go
git commit -m "Update permission mappings"
```

### Project Structure

```
cmd/least/              # CLI entry point
internal/
  provider/             # IaC provider abstraction
    terraform/          # Terraform HCL parser
    cloudformation/     # CloudFormation (stub)
  mapping/              # Resource → IAM action mappings
    gen/                # Code generator
    generated.go        # Generated from schemas
  policy/               # IAM policy generation
  checker/              # Policy comparison
  schema/               # CloudFormation schema handling
scripts/
  fetch-schemas.sh      # Download schemas from AWS
```

### Adding a New Provider

1. Create `internal/provider/<name>/<name>.go`
2. Implement the `provider.Provider` interface
3. Register in `cmd/least/main.go`

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
