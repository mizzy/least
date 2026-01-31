package mapping

import (
	"context"
	"sync"

	"github.com/mizzy/least/internal/schema"
)

// ResourceMapping defines IAM actions required for a Terraform resource type
type ResourceMapping struct {
	Create []string
	Read   []string
	Update []string
	Delete []string
}

// Resolver resolves IAM permissions for resource types
type Resolver struct {
	schemaStore   *schema.Store
	schemaFetcher *schema.Fetcher
	useSchema     bool
	mu            sync.RWMutex
}

// ResolverOption configures the Resolver
type ResolverOption func(*Resolver)

// WithSchemaStore enables schema-based permission resolution
func WithSchemaStore(store *schema.Store) ResolverOption {
	return func(r *Resolver) {
		r.schemaStore = store
		r.schemaFetcher = schema.NewFetcher(store)
		r.useSchema = true
	}
}

// NewResolver creates a new permission resolver
func NewResolver(opts ...ResolverOption) *Resolver {
	r := &Resolver{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// GetActionsForResourceType returns all IAM actions needed for a resource type
// It first tries CloudFormation schema, then falls back to hardcoded mappings
func (r *Resolver) GetActionsForResourceType(ctx context.Context, resourceType string) []string {
	// Try schema-based resolution first
	if r.useSchema && r.schemaStore != nil {
		cfnType := schema.TerraformToCfnType(resourceType)
		if cfnType != "" {
			if perms, err := r.schemaStore.GetPermissions(cfnType); err == nil {
				return perms.All
			}

			// Try fetching from AWS if CLI is available
			if r.schemaFetcher != nil && schema.IsAWSCLIAvailable() {
				if s, err := r.schemaFetcher.FetchSchema(ctx, cfnType); err == nil {
					if perms, _ := r.schemaStore.GetPermissions(cfnType); perms != nil {
						// Cache to file
						_ = r.schemaStore.SaveToCache(s)
						return perms.All
					}
				}
			}
		}
	}

	// Fallback to hardcoded mappings
	return GetActionsForResource(resourceType)
}

// GetActionsForResource returns all IAM actions needed for a resource type
// Uses hardcoded mappings (legacy function for backwards compatibility)
func GetActionsForResource(resourceType string) []string {
	mapping, ok := fallbackMappings[resourceType]
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
	types := make([]string, 0, len(fallbackMappings))
	for t := range fallbackMappings {
		types = append(types, t)
	}
	return types
}

// fallbackMappings contains hardcoded mappings for when schema is unavailable
// These serve as a fallback and for common resource types
var fallbackMappings = map[string]ResourceMapping{
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
		Create: []string{"s3:PutBucketVersioning"},
		Read:   []string{"s3:GetBucketVersioning"},
		Update: []string{"s3:PutBucketVersioning"},
		Delete: []string{},
	},
	"aws_s3_bucket_public_access_block": {
		Create: []string{"s3:PutBucketPublicAccessBlock"},
		Read:   []string{"s3:GetBucketPublicAccessBlock"},
		Update: []string{"s3:PutBucketPublicAccessBlock"},
		Delete: []string{"s3:PutBucketPublicAccessBlock"},
	},
	"aws_instance": {
		Create: []string{"ec2:RunInstances", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeInstances", "ec2:DescribeTags"},
		Update: []string{"ec2:ModifyInstanceAttribute", "ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:TerminateInstances"},
	},
	"aws_vpc": {
		Create: []string{"ec2:CreateVpc", "ec2:CreateTags", "ec2:ModifyVpcAttribute"},
		Read:   []string{"ec2:DescribeVpcs", "ec2:DescribeVpcAttribute"},
		Update: []string{"ec2:ModifyVpcAttribute", "ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:DeleteVpc"},
	},
	"aws_subnet": {
		Create: []string{"ec2:CreateSubnet", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeSubnets"},
		Update: []string{"ec2:ModifySubnetAttribute", "ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:DeleteSubnet"},
	},
	"aws_security_group": {
		Create: []string{"ec2:CreateSecurityGroup", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeSecurityGroups"},
		Update: []string{
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:AuthorizeSecurityGroupEgress",
			"ec2:RevokeSecurityGroupIngress",
			"ec2:RevokeSecurityGroupEgress",
			"ec2:CreateTags",
			"ec2:DeleteTags",
		},
		Delete: []string{"ec2:DeleteSecurityGroup"},
	},
	"aws_iam_role": {
		Create: []string{"iam:CreateRole", "iam:TagRole"},
		Read:   []string{"iam:GetRole", "iam:ListRoleTags"},
		Update: []string{"iam:UpdateRole", "iam:UpdateAssumeRolePolicy", "iam:TagRole", "iam:UntagRole"},
		Delete: []string{"iam:DeleteRole"},
	},
	"aws_iam_policy": {
		Create: []string{"iam:CreatePolicy", "iam:TagPolicy"},
		Read:   []string{"iam:GetPolicy", "iam:GetPolicyVersion"},
		Update: []string{"iam:CreatePolicyVersion", "iam:DeletePolicyVersion", "iam:TagPolicy", "iam:UntagPolicy"},
		Delete: []string{"iam:DeletePolicy"},
	},
	"aws_lambda_function": {
		Create: []string{"lambda:CreateFunction", "lambda:TagResource", "iam:PassRole"},
		Read:   []string{"lambda:GetFunction", "lambda:GetFunctionConfiguration", "lambda:ListTags"},
		Update: []string{"lambda:UpdateFunctionCode", "lambda:UpdateFunctionConfiguration", "lambda:TagResource", "lambda:UntagResource"},
		Delete: []string{"lambda:DeleteFunction"},
	},
	"aws_dynamodb_table": {
		Create: []string{"dynamodb:CreateTable", "dynamodb:TagResource"},
		Read:   []string{"dynamodb:DescribeTable", "dynamodb:ListTagsOfResource"},
		Update: []string{"dynamodb:UpdateTable", "dynamodb:TagResource", "dynamodb:UntagResource"},
		Delete: []string{"dynamodb:DeleteTable"},
	},
	"aws_ecs_cluster": {
		Create: []string{"ecs:CreateCluster", "ecs:TagResource"},
		Read:   []string{"ecs:DescribeClusters", "ecs:ListTagsForResource"},
		Update: []string{"ecs:UpdateCluster", "ecs:TagResource", "ecs:UntagResource"},
		Delete: []string{"ecs:DeleteCluster"},
	},
	"aws_ecs_service": {
		Create: []string{"ecs:CreateService", "ecs:TagResource", "iam:PassRole"},
		Read:   []string{"ecs:DescribeServices", "ecs:ListTagsForResource"},
		Update: []string{"ecs:UpdateService", "ecs:TagResource", "ecs:UntagResource"},
		Delete: []string{"ecs:DeleteService"},
	},
	"aws_ecs_task_definition": {
		Create: []string{"ecs:RegisterTaskDefinition", "ecs:TagResource", "iam:PassRole"},
		Read:   []string{"ecs:DescribeTaskDefinition", "ecs:ListTagsForResource"},
		Update: []string{"ecs:RegisterTaskDefinition", "ecs:TagResource", "ecs:UntagResource"},
		Delete: []string{"ecs:DeregisterTaskDefinition"},
	},
	"aws_rds_cluster": {
		Create: []string{"rds:CreateDBCluster", "rds:AddTagsToResource"},
		Read:   []string{"rds:DescribeDBClusters", "rds:ListTagsForResource"},
		Update: []string{"rds:ModifyDBCluster", "rds:AddTagsToResource", "rds:RemoveTagsFromResource"},
		Delete: []string{"rds:DeleteDBCluster"},
	},
	"aws_db_instance": {
		Create: []string{"rds:CreateDBInstance", "rds:AddTagsToResource"},
		Read:   []string{"rds:DescribeDBInstances", "rds:ListTagsForResource"},
		Update: []string{"rds:ModifyDBInstance", "rds:AddTagsToResource", "rds:RemoveTagsFromResource"},
		Delete: []string{"rds:DeleteDBInstance"},
	},
	"aws_sns_topic": {
		Create: []string{"sns:CreateTopic", "sns:TagResource"},
		Read:   []string{"sns:GetTopicAttributes", "sns:ListTagsForResource"},
		Update: []string{"sns:SetTopicAttributes", "sns:TagResource", "sns:UntagResource"},
		Delete: []string{"sns:DeleteTopic"},
	},
	"aws_sqs_queue": {
		Create: []string{"sqs:CreateQueue", "sqs:TagQueue"},
		Read:   []string{"sqs:GetQueueAttributes", "sqs:ListQueueTags"},
		Update: []string{"sqs:SetQueueAttributes", "sqs:TagQueue", "sqs:UntagQueue"},
		Delete: []string{"sqs:DeleteQueue"},
	},
	"aws_kms_key": {
		Create: []string{"kms:CreateKey", "kms:TagResource"},
		Read:   []string{"kms:DescribeKey", "kms:ListResourceTags"},
		Update: []string{"kms:UpdateKeyDescription", "kms:TagResource", "kms:UntagResource"},
		Delete: []string{"kms:ScheduleKeyDeletion"},
	},
	"aws_secretsmanager_secret": {
		Create: []string{"secretsmanager:CreateSecret", "secretsmanager:TagResource"},
		Read:   []string{"secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"},
		Update: []string{"secretsmanager:UpdateSecret", "secretsmanager:TagResource", "secretsmanager:UntagResource"},
		Delete: []string{"secretsmanager:DeleteSecret"},
	},
	"aws_ssm_parameter": {
		Create: []string{"ssm:PutParameter", "ssm:AddTagsToResource"},
		Read:   []string{"ssm:GetParameter", "ssm:GetParameters", "ssm:ListTagsForResource"},
		Update: []string{"ssm:PutParameter", "ssm:AddTagsToResource", "ssm:RemoveTagsFromResource"},
		Delete: []string{"ssm:DeleteParameter"},
	},
	"aws_cloudwatch_log_group": {
		Create: []string{"logs:CreateLogGroup", "logs:TagResource"},
		Read:   []string{"logs:DescribeLogGroups", "logs:ListTagsForResource"},
		Update: []string{"logs:PutRetentionPolicy", "logs:TagResource", "logs:UntagResource"},
		Delete: []string{"logs:DeleteLogGroup"},
	},
	"aws_ecr_repository": {
		Create: []string{"ecr:CreateRepository", "ecr:TagResource"},
		Read:   []string{"ecr:DescribeRepositories", "ecr:ListTagsForResource"},
		Update: []string{"ecr:TagResource", "ecr:UntagResource"},
		Delete: []string{"ecr:DeleteRepository"},
	},
	"aws_eks_cluster": {
		Create: []string{"eks:CreateCluster", "eks:TagResource", "iam:PassRole"},
		Read:   []string{"eks:DescribeCluster", "eks:ListTagsForResource"},
		Update: []string{"eks:UpdateClusterConfig", "eks:TagResource", "eks:UntagResource"},
		Delete: []string{"eks:DeleteCluster"},
	},
	"aws_lb": {
		Create: []string{"elasticloadbalancing:CreateLoadBalancer", "elasticloadbalancing:AddTags"},
		Read:   []string{"elasticloadbalancing:DescribeLoadBalancers", "elasticloadbalancing:DescribeTags"},
		Update: []string{"elasticloadbalancing:ModifyLoadBalancerAttributes", "elasticloadbalancing:AddTags", "elasticloadbalancing:RemoveTags"},
		Delete: []string{"elasticloadbalancing:DeleteLoadBalancer"},
	},
	"aws_lb_target_group": {
		Create: []string{"elasticloadbalancing:CreateTargetGroup", "elasticloadbalancing:AddTags"},
		Read:   []string{"elasticloadbalancing:DescribeTargetGroups", "elasticloadbalancing:DescribeTags"},
		Update: []string{"elasticloadbalancing:ModifyTargetGroup", "elasticloadbalancing:AddTags", "elasticloadbalancing:RemoveTags"},
		Delete: []string{"elasticloadbalancing:DeleteTargetGroup"},
	},
	"aws_api_gateway_rest_api": {
		Create: []string{"apigateway:POST"},
		Read:   []string{"apigateway:GET"},
		Update: []string{"apigateway:PATCH", "apigateway:PUT"},
		Delete: []string{"apigateway:DELETE"},
	},
	"aws_route53_zone": {
		Create: []string{"route53:CreateHostedZone", "route53:ChangeTagsForResource"},
		Read:   []string{"route53:GetHostedZone", "route53:ListTagsForResource"},
		Update: []string{"route53:UpdateHostedZoneComment", "route53:ChangeTagsForResource"},
		Delete: []string{"route53:DeleteHostedZone"},
	},
	"aws_route53_record": {
		Create: []string{"route53:ChangeResourceRecordSets"},
		Read:   []string{"route53:ListResourceRecordSets"},
		Update: []string{"route53:ChangeResourceRecordSets"},
		Delete: []string{"route53:ChangeResourceRecordSets"},
	},
	"aws_cloudfront_distribution": {
		Create: []string{"cloudfront:CreateDistribution", "cloudfront:TagResource"},
		Read:   []string{"cloudfront:GetDistribution", "cloudfront:ListTagsForResource"},
		Update: []string{"cloudfront:UpdateDistribution", "cloudfront:TagResource", "cloudfront:UntagResource"},
		Delete: []string{"cloudfront:DeleteDistribution"},
	},
	"aws_acm_certificate": {
		Create: []string{"acm:RequestCertificate", "acm:AddTagsToCertificate"},
		Read:   []string{"acm:DescribeCertificate", "acm:ListTagsForCertificate"},
		Update: []string{"acm:AddTagsToCertificate", "acm:RemoveTagsFromCertificate"},
		Delete: []string{"acm:DeleteCertificate"},
	},
	"aws_wafv2_web_acl": {
		Create: []string{"wafv2:CreateWebACL", "wafv2:TagResource"},
		Read:   []string{"wafv2:GetWebACL", "wafv2:ListTagsForResource"},
		Update: []string{"wafv2:UpdateWebACL", "wafv2:TagResource", "wafv2:UntagResource"},
		Delete: []string{"wafv2:DeleteWebACL"},
	},
	"aws_autoscaling_group": {
		Create: []string{"autoscaling:CreateAutoScalingGroup", "autoscaling:CreateOrUpdateTags"},
		Read:   []string{"autoscaling:DescribeAutoScalingGroups", "autoscaling:DescribeTags"},
		Update: []string{"autoscaling:UpdateAutoScalingGroup", "autoscaling:CreateOrUpdateTags", "autoscaling:DeleteTags"},
		Delete: []string{"autoscaling:DeleteAutoScalingGroup"},
	},
	"aws_sfn_state_machine": {
		Create: []string{"states:CreateStateMachine", "states:TagResource", "iam:PassRole"},
		Read:   []string{"states:DescribeStateMachine", "states:ListTagsForResource"},
		Update: []string{"states:UpdateStateMachine", "states:TagResource", "states:UntagResource"},
		Delete: []string{"states:DeleteStateMachine"},
	},
	"aws_glue_catalog_database": {
		Create: []string{"glue:CreateDatabase"},
		Read:   []string{"glue:GetDatabase"},
		Update: []string{"glue:UpdateDatabase"},
		Delete: []string{"glue:DeleteDatabase"},
	},
	"aws_kinesis_stream": {
		Create: []string{"kinesis:CreateStream", "kinesis:AddTagsToStream"},
		Read:   []string{"kinesis:DescribeStream", "kinesis:ListTagsForStream"},
		Update: []string{"kinesis:UpdateShardCount", "kinesis:AddTagsToStream", "kinesis:RemoveTagsFromStream"},
		Delete: []string{"kinesis:DeleteStream"},
	},
	"aws_cognito_user_pool": {
		Create: []string{"cognito-idp:CreateUserPool", "cognito-idp:TagResource"},
		Read:   []string{"cognito-idp:DescribeUserPool", "cognito-idp:ListTagsForResource"},
		Update: []string{"cognito-idp:UpdateUserPool", "cognito-idp:TagResource", "cognito-idp:UntagResource"},
		Delete: []string{"cognito-idp:DeleteUserPool"},
	},
	"aws_elasticache_cluster": {
		Create: []string{"elasticache:CreateCacheCluster", "elasticache:AddTagsToResource"},
		Read:   []string{"elasticache:DescribeCacheClusters", "elasticache:ListTagsForResource"},
		Update: []string{"elasticache:ModifyCacheCluster", "elasticache:AddTagsToResource", "elasticache:RemoveTagsFromResource"},
		Delete: []string{"elasticache:DeleteCacheCluster"},
	},
	"aws_internet_gateway": {
		Create: []string{"ec2:CreateInternetGateway", "ec2:AttachInternetGateway", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeInternetGateways"},
		Update: []string{"ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:DetachInternetGateway", "ec2:DeleteInternetGateway"},
	},
	"aws_nat_gateway": {
		Create: []string{"ec2:CreateNatGateway", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeNatGateways"},
		Update: []string{"ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:DeleteNatGateway"},
	},
	"aws_route_table": {
		Create: []string{"ec2:CreateRouteTable", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeRouteTables"},
		Update: []string{"ec2:CreateRoute", "ec2:DeleteRoute", "ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:DeleteRouteTable"},
	},
	"aws_eip": {
		Create: []string{"ec2:AllocateAddress", "ec2:CreateTags"},
		Read:   []string{"ec2:DescribeAddresses"},
		Update: []string{"ec2:CreateTags", "ec2:DeleteTags"},
		Delete: []string{"ec2:ReleaseAddress"},
	},
}
