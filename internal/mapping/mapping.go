package mapping

// ResourceMapping defines IAM actions required for a Terraform resource type
type ResourceMapping struct {
	// Actions required to create the resource
	Create []string
	// Actions required to read/describe the resource
	Read []string
	// Actions required to update the resource
	Update []string
	// Actions required to delete the resource
	Delete []string
}

// AWSMappings maps Terraform AWS resource types to required IAM actions
// This is the core mapping table that drives policy generation
var AWSMappings = map[string]ResourceMapping{
	"aws_s3_bucket": {
		Create: []string{
			"s3:CreateBucket",
			"s3:PutBucketTagging",
		},
		Read: []string{
			"s3:GetBucket*",
			"s3:ListBucket",
		},
		Update: []string{
			"s3:PutBucket*",
		},
		Delete: []string{
			"s3:DeleteBucket",
		},
	},
	"aws_s3_bucket_versioning": {
		Create: []string{
			"s3:PutBucketVersioning",
		},
		Read: []string{
			"s3:GetBucketVersioning",
		},
		Update: []string{
			"s3:PutBucketVersioning",
		},
		Delete: []string{},
	},
	"aws_s3_bucket_public_access_block": {
		Create: []string{
			"s3:PutBucketPublicAccessBlock",
		},
		Read: []string{
			"s3:GetBucketPublicAccessBlock",
		},
		Update: []string{
			"s3:PutBucketPublicAccessBlock",
		},
		Delete: []string{
			"s3:PutBucketPublicAccessBlock",
		},
	},
	"aws_instance": {
		Create: []string{
			"ec2:RunInstances",
			"ec2:CreateTags",
		},
		Read: []string{
			"ec2:DescribeInstances",
			"ec2:DescribeTags",
		},
		Update: []string{
			"ec2:ModifyInstanceAttribute",
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Delete: []string{
			"ec2:TerminateInstances",
		},
	},
	"aws_vpc": {
		Create: []string{
			"ec2:CreateVpc",
			"ec2:CreateTags",
			"ec2:ModifyVpcAttribute",
		},
		Read: []string{
			"ec2:DescribeVpcs",
			"ec2:DescribeVpcAttribute",
		},
		Update: []string{
			"ec2:ModifyVpcAttribute",
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Delete: []string{
			"ec2:DeleteVpc",
		},
	},
	"aws_subnet": {
		Create: []string{
			"ec2:CreateSubnet",
			"ec2:CreateTags",
		},
		Read: []string{
			"ec2:DescribeSubnets",
		},
		Update: []string{
			"ec2:ModifySubnetAttribute",
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Delete: []string{
			"ec2:DeleteSubnet",
		},
	},
	"aws_security_group": {
		Create: []string{
			"ec2:CreateSecurityGroup",
			"ec2:CreateTags",
		},
		Read: []string{
			"ec2:DescribeSecurityGroups",
		},
		Update: []string{
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:AuthorizeSecurityGroupEgress",
			"ec2:RevokeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupEgress",
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Delete: []string{
			"ec2:DeleteSecurityGroup",
		},
	},
	"aws_iam_role": {
		Create: []string{
			"iam:CreateRole",
			"iam:TagRole",
		},
		Read: []string{
			"iam:GetRole",
			"iam:ListRoleTags",
		},
		Update: []string{
			"iam:UpdateRole",
			"iam:UpdateAssumeRolePolicy",
			"iam:TagRole",
			"iam:UntagRole",
		},
		Delete: []string{
			"iam:DeleteRole",
		},
	},
	"aws_iam_policy": {
		Create: []string{
			"iam:CreatePolicy",
			"iam:TagPolicy",
		},
		Read: []string{
			"iam:GetPolicy",
			"iam:GetPolicyVersion",
		},
		Update: []string{
			"iam:CreatePolicyVersion",
			"iam:DeletePolicyVersion",
			"iam:TagPolicy",
			"iam:UntagPolicy",
		},
		Delete: []string{
			"iam:DeletePolicy",
		},
	},
	"aws_lambda_function": {
		Create: []string{
			"lambda:CreateFunction",
			"lambda:TagResource",
			"iam:PassRole",
		},
		Read: []string{
			"lambda:GetFunction",
			"lambda:GetFunctionConfiguration",
			"lambda:ListTags",
		},
		Update: []string{
			"lambda:UpdateFunctionCode",
			"lambda:UpdateFunctionConfiguration",
			"lambda:TagResource",
			"lambda:UntagResource",
		},
		Delete: []string{
			"lambda:DeleteFunction",
		},
	},
	"aws_dynamodb_table": {
		Create: []string{
			"dynamodb:CreateTable",
			"dynamodb:TagResource",
		},
		Read: []string{
			"dynamodb:DescribeTable",
			"dynamodb:ListTagsOfResource",
		},
		Update: []string{
			"dynamodb:UpdateTable",
			"dynamodb:TagResource",
			"dynamodb:UntagResource",
		},
		Delete: []string{
			"dynamodb:DeleteTable",
		},
	},
}

// GetActionsForResource returns all IAM actions needed for a resource type
func GetActionsForResource(resourceType string) []string {
	mapping, ok := AWSMappings[resourceType]
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var actions []string

	for _, actionList := range [][]string{mapping.Create, mapping.Read, mapping.Update, mapping.Delete} {
		for _, action := range actionList {
			if !seen[action] {
				seen[action] = true
				actions = append(actions, action)
			}
		}
	}

	return actions
}

// GetSupportedResourceTypes returns list of supported Terraform resource types
func GetSupportedResourceTypes() []string {
	types := make([]string, 0, len(AWSMappings))
	for t := range AWSMappings {
		types = append(types, t)
	}
	return types
}
