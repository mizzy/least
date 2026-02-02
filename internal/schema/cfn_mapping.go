package schema

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TerraformToCfnType converts a Terraform resource type to CloudFormation type
// e.g., "aws_s3_bucket" -> "AWS::S3::Bucket"
func TerraformToCfnType(tfType string) string {
	// Check explicit mappings first
	if cfnType, ok := tfToCfnMappings[tfType]; ok {
		return cfnType
	}

	// Try automatic conversion for aws_ prefixed resources
	if !strings.HasPrefix(tfType, "aws_") {
		return ""
	}

	// aws_s3_bucket -> S3::Bucket
	parts := strings.Split(strings.TrimPrefix(tfType, "aws_"), "_")
	if len(parts) < 2 {
		return ""
	}

	// First part is service name
	service := strings.ToUpper(parts[0])

	// Rest is resource name in PascalCase
	caser := cases.Title(language.English)
	var resourceParts []string
	for _, p := range parts[1:] {
		resourceParts = append(resourceParts, caser.String(p))
	}
	resource := strings.Join(resourceParts, "")

	return "AWS::" + service + "::" + resource
}

// CfnToTerraformType converts a CloudFormation type to Terraform resource type
// e.g., "AWS::S3::Bucket" -> "aws_s3_bucket"
func CfnToTerraformType(cfnType string) string {
	// Check explicit mappings first
	if tfType, ok := cfnToTfMappings[cfnType]; ok {
		return tfType
	}

	// Try automatic conversion
	if !strings.HasPrefix(cfnType, "AWS::") {
		return ""
	}

	// AWS::S3::Bucket -> aws_s3_bucket
	parts := strings.Split(strings.TrimPrefix(cfnType, "AWS::"), "::")
	if len(parts) != 2 {
		return ""
	}

	service := strings.ToLower(parts[0])
	resource := toSnakeCase(parts[1])

	return "aws_" + service + "_" + resource
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// Explicit mappings where automatic conversion doesn't work
var tfToCfnMappings = map[string]string{
	// EC2
	"aws_instance":            "AWS::EC2::Instance",
	"aws_eip":                 "AWS::EC2::EIP",
	"aws_eip_association":     "AWS::EC2::EIPAssociation",
	"aws_security_group":      "AWS::EC2::SecurityGroup",
	"aws_security_group_rule": "AWS::EC2::SecurityGroupIngress", // partial mapping
	"aws_network_interface":   "AWS::EC2::NetworkInterface",
	"aws_key_pair":            "AWS::EC2::KeyPair",
	"aws_launch_template":     "AWS::EC2::LaunchTemplate",
	"aws_placement_group":     "AWS::EC2::PlacementGroup",

	// VPC
	"aws_vpc":                     "AWS::EC2::VPC",
	"aws_subnet":                  "AWS::EC2::Subnet",
	"aws_internet_gateway":        "AWS::EC2::InternetGateway",
	"aws_nat_gateway":             "AWS::EC2::NatGateway",
	"aws_route_table":             "AWS::EC2::RouteTable",
	"aws_route":                   "AWS::EC2::Route",
	"aws_route_table_association": "AWS::EC2::SubnetRouteTableAssociation",
	"aws_vpc_endpoint":            "AWS::EC2::VPCEndpoint",
	"aws_vpc_peering_connection":  "AWS::EC2::VPCPeeringConnection",
	"aws_network_acl":             "AWS::EC2::NetworkAcl",
	"aws_network_acl_rule":        "AWS::EC2::NetworkAclEntry",

	// S3
	"aws_s3_bucket":                         "AWS::S3::Bucket",
	"aws_s3_bucket_policy":                  "AWS::S3::BucketPolicy",
	"aws_s3_bucket_versioning":              "AWS::S3::Bucket", // sub-resource
	"aws_s3_bucket_acl":                     "AWS::S3::Bucket",
	"aws_s3_bucket_cors_configuration":      "AWS::S3::Bucket",
	"aws_s3_bucket_lifecycle_configuration": "AWS::S3::Bucket",
	"aws_s3_bucket_logging":                 "AWS::S3::Bucket",
	"aws_s3_bucket_public_access_block":     "AWS::S3::Bucket",

	// IAM
	"aws_iam_role":                    "AWS::IAM::Role",
	"aws_iam_policy":                  "AWS::IAM::ManagedPolicy",
	"aws_iam_role_policy":             "AWS::IAM::Policy",
	"aws_iam_role_policy_attachment":  "AWS::IAM::Role", // handled via role
	"aws_iam_user":                    "AWS::IAM::User",
	"aws_iam_user_policy":             "AWS::IAM::Policy",
	"aws_iam_user_policy_attachment":  "AWS::IAM::User",
	"aws_iam_group":                   "AWS::IAM::Group",
	"aws_iam_group_policy":            "AWS::IAM::Policy",
	"aws_iam_group_policy_attachment": "AWS::IAM::Group",
	"aws_iam_instance_profile":        "AWS::IAM::InstanceProfile",

	// Lambda
	"aws_lambda_function":             "AWS::Lambda::Function",
	"aws_lambda_permission":           "AWS::Lambda::Permission",
	"aws_lambda_event_source_mapping": "AWS::Lambda::EventSourceMapping",
	"aws_lambda_layer_version":        "AWS::Lambda::LayerVersion",
	"aws_lambda_alias":                "AWS::Lambda::Alias",

	// DynamoDB
	"aws_dynamodb_table":        "AWS::DynamoDB::Table",
	"aws_dynamodb_global_table": "AWS::DynamoDB::GlobalTable",

	// RDS
	"aws_db_instance":                "AWS::RDS::DBInstance",
	"aws_db_cluster":                 "AWS::RDS::DBCluster",
	"aws_db_subnet_group":            "AWS::RDS::DBSubnetGroup",
	"aws_db_parameter_group":         "AWS::RDS::DBParameterGroup",
	"aws_db_cluster_parameter_group": "AWS::RDS::DBClusterParameterGroup",
	"aws_rds_cluster":                "AWS::RDS::DBCluster",

	// ECS
	"aws_ecs_cluster":         "AWS::ECS::Cluster",
	"aws_ecs_service":         "AWS::ECS::Service",
	"aws_ecs_task_definition": "AWS::ECS::TaskDefinition",

	// EKS
	"aws_eks_cluster":    "AWS::EKS::Cluster",
	"aws_eks_node_group": "AWS::EKS::Nodegroup",
	"aws_eks_addon":      "AWS::EKS::Addon",

	// CloudWatch
	"aws_cloudwatch_log_group":    "AWS::Logs::LogGroup",
	"aws_cloudwatch_log_stream":   "AWS::Logs::LogStream",
	"aws_cloudwatch_metric_alarm": "AWS::CloudWatch::Alarm",
	"aws_cloudwatch_dashboard":    "AWS::CloudWatch::Dashboard",

	// SNS/SQS
	"aws_sns_topic":              "AWS::SNS::Topic",
	"aws_sns_topic_policy":       "AWS::SNS::TopicPolicy",
	"aws_sns_topic_subscription": "AWS::SNS::Subscription",
	"aws_sqs_queue":              "AWS::SQS::Queue",
	"aws_sqs_queue_policy":       "AWS::SQS::QueuePolicy",

	// API Gateway
	"aws_api_gateway_rest_api":     "AWS::ApiGateway::RestApi",
	"aws_api_gateway_resource":     "AWS::ApiGateway::Resource",
	"aws_api_gateway_method":       "AWS::ApiGateway::Method",
	"aws_api_gateway_integration":  "AWS::ApiGateway::Method",
	"aws_api_gateway_deployment":   "AWS::ApiGateway::Deployment",
	"aws_api_gateway_stage":        "AWS::ApiGateway::Stage",
	"aws_apigatewayv2_api":         "AWS::ApiGatewayV2::Api",
	"aws_apigatewayv2_stage":       "AWS::ApiGatewayV2::Stage",
	"aws_apigatewayv2_route":       "AWS::ApiGatewayV2::Route",
	"aws_apigatewayv2_integration": "AWS::ApiGatewayV2::Integration",

	// Secrets Manager / SSM
	"aws_secretsmanager_secret":         "AWS::SecretsManager::Secret",
	"aws_secretsmanager_secret_version": "AWS::SecretsManager::Secret",
	"aws_ssm_parameter":                 "AWS::SSM::Parameter",

	// KMS
	"aws_kms_key":   "AWS::KMS::Key",
	"aws_kms_alias": "AWS::KMS::Alias",

	// ACM
	"aws_acm_certificate":            "AWS::CertificateManager::Certificate",
	"aws_acm_certificate_validation": "AWS::CertificateManager::Certificate",

	// Route53
	"aws_route53_zone":   "AWS::Route53::HostedZone",
	"aws_route53_record": "AWS::Route53::RecordSet",

	// CloudFront
	"aws_cloudfront_distribution":           "AWS::CloudFront::Distribution",
	"aws_cloudfront_origin_access_identity": "AWS::CloudFront::CloudFrontOriginAccessIdentity",

	// Elasticache
	"aws_elasticache_cluster":           "AWS::ElastiCache::CacheCluster",
	"aws_elasticache_replication_group": "AWS::ElastiCache::ReplicationGroup",
	"aws_elasticache_subnet_group":      "AWS::ElastiCache::SubnetGroup",

	// ELB
	"aws_lb":               "AWS::ElasticLoadBalancingV2::LoadBalancer",
	"aws_alb":              "AWS::ElasticLoadBalancingV2::LoadBalancer",
	"aws_lb_target_group":  "AWS::ElasticLoadBalancingV2::TargetGroup",
	"aws_alb_target_group": "AWS::ElasticLoadBalancingV2::TargetGroup",
	"aws_lb_listener":      "AWS::ElasticLoadBalancingV2::Listener",
	"aws_alb_listener":     "AWS::ElasticLoadBalancingV2::Listener",
	"aws_lb_listener_rule": "AWS::ElasticLoadBalancingV2::ListenerRule",

	// Step Functions
	"aws_sfn_state_machine": "AWS::StepFunctions::StateMachine",
	"aws_sfn_activity":      "AWS::StepFunctions::Activity",

	// EventBridge
	"aws_cloudwatch_event_rule":   "AWS::Events::Rule",
	"aws_cloudwatch_event_target": "AWS::Events::Rule",

	// CodeBuild/CodePipeline
	"aws_codebuild_project": "AWS::CodeBuild::Project",
	"aws_codepipeline":      "AWS::CodePipeline::Pipeline",

	// Cognito
	"aws_cognito_user_pool":        "AWS::Cognito::UserPool",
	"aws_cognito_user_pool_client": "AWS::Cognito::UserPoolClient",
	"aws_cognito_identity_pool":    "AWS::Cognito::IdentityPool",

	// Kinesis
	"aws_kinesis_stream":                   "AWS::Kinesis::Stream",
	"aws_kinesis_firehose_delivery_stream": "AWS::KinesisFirehose::DeliveryStream",

	// Glue
	"aws_glue_catalog_database": "AWS::Glue::Database",
	"aws_glue_catalog_table":    "AWS::Glue::Table",
	"aws_glue_crawler":          "AWS::Glue::Crawler",
	"aws_glue_job":              "AWS::Glue::Job",

	// Athena
	"aws_athena_workgroup": "AWS::Athena::WorkGroup",
	"aws_athena_database":  "AWS::Athena::Database",

	// ECR
	"aws_ecr_repository":        "AWS::ECR::Repository",
	"aws_ecr_lifecycle_policy":  "AWS::ECR::Repository",
	"aws_ecr_repository_policy": "AWS::ECR::Repository",

	// WAF
	"aws_wafv2_web_acl":    "AWS::WAFv2::WebACL",
	"aws_wafv2_ip_set":     "AWS::WAFv2::IPSet",
	"aws_wafv2_rule_group": "AWS::WAFv2::RuleGroup",

	// Auto Scaling
	"aws_autoscaling_group":    "AWS::AutoScaling::AutoScalingGroup",
	"aws_autoscaling_policy":   "AWS::AutoScaling::ScalingPolicy",
	"aws_autoscaling_schedule": "AWS::AutoScaling::ScheduledAction",
	"aws_launch_configuration": "AWS::AutoScaling::LaunchConfiguration",

	// CloudTrail
	"aws_cloudtrail": "AWS::CloudTrail::Trail",

	// Config
	"aws_config_config_rule":            "AWS::Config::ConfigRule",
	"aws_config_configuration_recorder": "AWS::Config::ConfigurationRecorder",

	// Backup
	"aws_backup_vault": "AWS::Backup::BackupVault",
	"aws_backup_plan":  "AWS::Backup::BackupPlan",
}

// Reverse mapping
var cfnToTfMappings = func() map[string]string {
	m := make(map[string]string)
	for tf, cfn := range tfToCfnMappings {
		// Only add if not already present (prefer first mapping)
		if _, ok := m[cfn]; !ok {
			m[cfn] = tf
		}
	}
	return m
}()
