# Pattern: No resources - only variables and outputs
variable "environment" {
  type    = string
  default = "dev"
}

variable "region" {
  type    = string
  default = "us-east-1"
}

output "config" {
  value = {
    environment = var.environment
    region      = var.region
  }
}

locals {
  common_tags = {
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}
