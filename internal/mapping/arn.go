package mapping

// ARNPattern defines how to construct an ARN for a resource type
type ARNPattern struct {
	// Pattern is the ARN template with placeholders: {account}, {region}, {attribute_name}
	Pattern string
	// ResourceAttribute is the Terraform attribute name used in the ARN (e.g., "bucket", "function_name")
	ResourceAttribute string
	// ChildPatterns are additional ARN patterns for child resources (e.g., S3 objects)
	ChildPatterns []string
}

// ARNPatterns maps Terraform resource types to their ARN patterns
var ARNPatterns = map[string]ARNPattern{
	// S3
	"aws_s3_bucket": {
		Pattern:           "arn:aws:s3:::{bucket}",
		ResourceAttribute: "bucket",
		ChildPatterns:     []string{"arn:aws:s3:::{bucket}/*"},
	},
	"aws_s3_bucket_versioning": {
		Pattern:           "arn:aws:s3:::{bucket}",
		ResourceAttribute: "bucket",
	},
	"aws_s3_bucket_public_access_block": {
		Pattern:           "arn:aws:s3:::{bucket}",
		ResourceAttribute: "bucket",
	},

	// Lambda
	"aws_lambda_function": {
		Pattern:           "arn:aws:lambda:{region}:{account}:function:{function_name}",
		ResourceAttribute: "function_name",
	},

	// DynamoDB
	"aws_dynamodb_table": {
		Pattern:           "arn:aws:dynamodb:{region}:{account}:table/{name}",
		ResourceAttribute: "name",
	},

	// IAM
	"aws_iam_role": {
		Pattern:           "arn:aws:iam::{account}:role/{name}",
		ResourceAttribute: "name",
	},
	"aws_iam_policy": {
		Pattern:           "arn:aws:iam::{account}:policy/{name}",
		ResourceAttribute: "name",
	},

	// RDS
	"aws_db_instance": {
		Pattern:           "arn:aws:rds:{region}:{account}:db:{identifier}",
		ResourceAttribute: "identifier",
	},
	"aws_rds_cluster": {
		Pattern:           "arn:aws:rds:{region}:{account}:cluster:{cluster_identifier}",
		ResourceAttribute: "cluster_identifier",
	},

	// SQS
	"aws_sqs_queue": {
		Pattern:           "arn:aws:sqs:{region}:{account}:{name}",
		ResourceAttribute: "name",
	},

	// SNS
	"aws_sns_topic": {
		Pattern:           "arn:aws:sns:{region}:{account}:{name}",
		ResourceAttribute: "name",
	},

	// KMS
	"aws_kms_key": {
		Pattern:           "arn:aws:kms:{region}:{account}:key/*",
		ResourceAttribute: "", // key_id is dynamically generated
	},

	// Secrets Manager
	"aws_secretsmanager_secret": {
		Pattern:           "arn:aws:secretsmanager:{region}:{account}:secret:{name}*",
		ResourceAttribute: "name",
	},

	// SSM
	"aws_ssm_parameter": {
		Pattern:           "arn:aws:ssm:{region}:{account}:parameter/{name}",
		ResourceAttribute: "name",
	},

	// CloudWatch Logs
	"aws_cloudwatch_log_group": {
		Pattern:           "arn:aws:logs:{region}:{account}:log-group:{name}",
		ResourceAttribute: "name",
	},

	// ECR
	"aws_ecr_repository": {
		Pattern:           "arn:aws:ecr:{region}:{account}:repository/{name}",
		ResourceAttribute: "name",
	},

	// ECS
	"aws_ecs_cluster": {
		Pattern:           "arn:aws:ecs:{region}:{account}:cluster/{name}",
		ResourceAttribute: "name",
	},
	"aws_ecs_service": {
		Pattern:           "arn:aws:ecs:{region}:{account}:service/{cluster}/*",
		ResourceAttribute: "cluster",
	},
	"aws_ecs_task_definition": {
		Pattern:           "arn:aws:ecs:{region}:{account}:task-definition/{family}:*",
		ResourceAttribute: "family",
	},

	// EKS
	"aws_eks_cluster": {
		Pattern:           "arn:aws:eks:{region}:{account}:cluster/{name}",
		ResourceAttribute: "name",
	},

	// Kinesis
	"aws_kinesis_stream": {
		Pattern:           "arn:aws:kinesis:{region}:{account}:stream/{name}",
		ResourceAttribute: "name",
	},

	// Cognito
	"aws_cognito_user_pool": {
		Pattern:           "arn:aws:cognito-idp:{region}:{account}:userpool/*",
		ResourceAttribute: "", // user pool id is dynamically generated
	},

	// ElastiCache
	"aws_elasticache_cluster": {
		Pattern:           "arn:aws:elasticache:{region}:{account}:cluster:{cluster_id}",
		ResourceAttribute: "cluster_id",
	},

	// Step Functions
	"aws_sfn_state_machine": {
		Pattern:           "arn:aws:states:{region}:{account}:stateMachine:{name}",
		ResourceAttribute: "name",
	},

	// Glue
	"aws_glue_catalog_database": {
		Pattern:           "arn:aws:glue:{region}:{account}:database/{name}",
		ResourceAttribute: "name",
	},

	// Route53
	"aws_route53_zone": {
		Pattern:           "arn:aws:route53:::hostedzone/*",
		ResourceAttribute: "", // zone_id is dynamically generated
	},
	"aws_route53_record": {
		Pattern:           "arn:aws:route53:::hostedzone/*",
		ResourceAttribute: "",
	},

	// CloudFront
	"aws_cloudfront_distribution": {
		Pattern:           "arn:aws:cloudfront::{account}:distribution/*",
		ResourceAttribute: "", // distribution_id is dynamically generated
	},

	// ACM
	"aws_acm_certificate": {
		Pattern:           "arn:aws:acm:{region}:{account}:certificate/*",
		ResourceAttribute: "", // certificate_arn is dynamically generated
	},

	// WAFv2
	"aws_wafv2_web_acl": {
		Pattern:           "arn:aws:wafv2:{region}:{account}:*/webacl/{name}/*",
		ResourceAttribute: "name",
	},

	// Auto Scaling
	"aws_autoscaling_group": {
		Pattern:           "arn:aws:autoscaling:{region}:{account}:autoScalingGroup:*:autoScalingGroupName/{name}",
		ResourceAttribute: "name",
	},

	// ELB
	"aws_lb": {
		Pattern:           "arn:aws:elasticloadbalancing:{region}:{account}:loadbalancer/*/{name}/*",
		ResourceAttribute: "name",
	},
	"aws_lb_target_group": {
		Pattern:           "arn:aws:elasticloadbalancing:{region}:{account}:targetgroup/{name}/*",
		ResourceAttribute: "name",
	},

	// API Gateway
	"aws_api_gateway_rest_api": {
		Pattern:           "arn:aws:apigateway:{region}::/restapis/*",
		ResourceAttribute: "", // rest_api_id is dynamically generated
	},

	// EC2 - IDs are dynamically generated, so use wildcards
	"aws_instance": {
		Pattern:           "arn:aws:ec2:{region}:{account}:instance/*",
		ResourceAttribute: "",
	},
	"aws_vpc": {
		Pattern:           "arn:aws:ec2:{region}:{account}:vpc/*",
		ResourceAttribute: "",
	},
	"aws_subnet": {
		Pattern:           "arn:aws:ec2:{region}:{account}:subnet/*",
		ResourceAttribute: "",
	},
	"aws_security_group": {
		Pattern:           "arn:aws:ec2:{region}:{account}:security-group/*",
		ResourceAttribute: "",
	},
	"aws_internet_gateway": {
		Pattern:           "arn:aws:ec2:{region}:{account}:internet-gateway/*",
		ResourceAttribute: "",
	},
	"aws_nat_gateway": {
		Pattern:           "arn:aws:ec2:{region}:{account}:natgateway/*",
		ResourceAttribute: "",
	},
	"aws_route_table": {
		Pattern:           "arn:aws:ec2:{region}:{account}:route-table/*",
		ResourceAttribute: "",
	},
	"aws_eip": {
		Pattern:           "arn:aws:ec2:{region}:{account}:elastic-ip/*",
		ResourceAttribute: "",
	},
}

// GetARNPattern returns the ARN pattern for a given resource type
func GetARNPattern(resourceType string) (ARNPattern, bool) {
	p, ok := ARNPatterns[resourceType]
	return p, ok
}

// GetARNAttributes returns the attribute names needed to construct the ARN for a resource type
func GetARNAttributes(resourceType string) []string {
	if p, ok := ARNPatterns[resourceType]; ok && p.ResourceAttribute != "" {
		return []string{p.ResourceAttribute}
	}
	return nil
}
