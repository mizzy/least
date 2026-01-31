#!/bin/bash
# Fetch CloudFormation resource schemas from AWS
# Run this script periodically to update the embedded schemas
#
# Prerequisites:
#   - AWS CLI configured with valid credentials
#   - jq installed
#
# Usage:
#   ./scripts/fetch-schemas.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_DIR="${SCRIPT_DIR}/../internal/schema/data"

mkdir -p "$SCHEMA_DIR"

# List of CloudFormation resource types to fetch
# These correspond to the most commonly used Terraform AWS resources
RESOURCE_TYPES=(
    # S3
    "AWS::S3::Bucket"
    "AWS::S3::BucketPolicy"

    # EC2
    "AWS::EC2::Instance"
    "AWS::EC2::VPC"
    "AWS::EC2::Subnet"
    "AWS::EC2::SecurityGroup"
    "AWS::EC2::InternetGateway"
    "AWS::EC2::NatGateway"
    "AWS::EC2::RouteTable"
    "AWS::EC2::EIP"
    "AWS::EC2::NetworkInterface"
    "AWS::EC2::LaunchTemplate"

    # IAM
    "AWS::IAM::Role"
    "AWS::IAM::ManagedPolicy"
    "AWS::IAM::Policy"
    "AWS::IAM::User"
    "AWS::IAM::Group"
    "AWS::IAM::InstanceProfile"

    # Lambda
    "AWS::Lambda::Function"
    "AWS::Lambda::Permission"
    "AWS::Lambda::EventSourceMapping"
    "AWS::Lambda::LayerVersion"

    # DynamoDB
    "AWS::DynamoDB::Table"
    "AWS::DynamoDB::GlobalTable"

    # RDS
    "AWS::RDS::DBInstance"
    "AWS::RDS::DBCluster"
    "AWS::RDS::DBSubnetGroup"
    "AWS::RDS::DBParameterGroup"

    # ECS
    "AWS::ECS::Cluster"
    "AWS::ECS::Service"
    "AWS::ECS::TaskDefinition"

    # EKS
    "AWS::EKS::Cluster"
    "AWS::EKS::Nodegroup"
    "AWS::EKS::Addon"

    # CloudWatch / Logs
    "AWS::Logs::LogGroup"
    "AWS::CloudWatch::Alarm"
    "AWS::CloudWatch::Dashboard"

    # SNS / SQS
    "AWS::SNS::Topic"
    "AWS::SNS::TopicPolicy"
    "AWS::SNS::Subscription"
    "AWS::SQS::Queue"
    "AWS::SQS::QueuePolicy"

    # API Gateway
    "AWS::ApiGateway::RestApi"
    "AWS::ApiGateway::Resource"
    "AWS::ApiGateway::Method"
    "AWS::ApiGateway::Deployment"
    "AWS::ApiGateway::Stage"
    "AWS::ApiGatewayV2::Api"

    # Secrets / SSM
    "AWS::SecretsManager::Secret"
    "AWS::SSM::Parameter"

    # KMS
    "AWS::KMS::Key"
    "AWS::KMS::Alias"

    # ACM
    "AWS::CertificateManager::Certificate"

    # Route53
    "AWS::Route53::HostedZone"
    "AWS::Route53::RecordSet"

    # CloudFront
    "AWS::CloudFront::Distribution"

    # ElastiCache
    "AWS::ElastiCache::CacheCluster"
    "AWS::ElastiCache::ReplicationGroup"
    "AWS::ElastiCache::SubnetGroup"

    # ELB
    "AWS::ElasticLoadBalancingV2::LoadBalancer"
    "AWS::ElasticLoadBalancingV2::TargetGroup"
    "AWS::ElasticLoadBalancingV2::Listener"
    "AWS::ElasticLoadBalancingV2::ListenerRule"

    # Step Functions
    "AWS::StepFunctions::StateMachine"

    # EventBridge
    "AWS::Events::Rule"

    # CodeBuild / CodePipeline
    "AWS::CodeBuild::Project"
    "AWS::CodePipeline::Pipeline"

    # Cognito
    "AWS::Cognito::UserPool"
    "AWS::Cognito::UserPoolClient"
    "AWS::Cognito::IdentityPool"

    # Kinesis
    "AWS::Kinesis::Stream"
    "AWS::KinesisFirehose::DeliveryStream"

    # ECR
    "AWS::ECR::Repository"

    # WAF
    "AWS::WAFv2::WebACL"
    "AWS::WAFv2::IPSet"
    "AWS::WAFv2::RuleGroup"

    # Auto Scaling
    "AWS::AutoScaling::AutoScalingGroup"
    "AWS::AutoScaling::LaunchConfiguration"
    "AWS::AutoScaling::ScalingPolicy"

    # CloudTrail
    "AWS::CloudTrail::Trail"

    # Glue
    "AWS::Glue::Database"
    "AWS::Glue::Table"
    "AWS::Glue::Crawler"
    "AWS::Glue::Job"

    # Athena
    "AWS::Athena::WorkGroup"

    # Backup
    "AWS::Backup::BackupVault"
    "AWS::Backup::BackupPlan"
)

echo "Fetching ${#RESOURCE_TYPES[@]} CloudFormation resource schemas..."
echo ""

SUCCESS=0
FAILED=0

for TYPE in "${RESOURCE_TYPES[@]}"; do
    # Convert AWS::S3::Bucket to aws-s3-bucket.json
    FILENAME=$(echo "$TYPE" | tr '[:upper:]' '[:lower:]' | sed 's/::/-/g').json
    FILEPATH="${SCHEMA_DIR}/${FILENAME}"

    echo -n "Fetching $TYPE... "

    if aws cloudformation describe-type \
        --type RESOURCE \
        --type-name "$TYPE" \
        --output json 2>/dev/null | jq -r '.Schema' > "$FILEPATH.tmp" 2>/dev/null; then

        # Validate JSON and format
        if jq . "$FILEPATH.tmp" > "$FILEPATH" 2>/dev/null; then
            rm "$FILEPATH.tmp"
            echo "OK"
            ((SUCCESS++))
        else
            rm -f "$FILEPATH.tmp" "$FILEPATH"
            echo "FAILED (invalid JSON)"
            ((FAILED++))
        fi
    else
        rm -f "$FILEPATH.tmp"
        echo "FAILED"
        ((FAILED++))
    fi
done

echo ""
echo "Done: $SUCCESS succeeded, $FAILED failed"
echo "Schemas saved to: $SCHEMA_DIR"
