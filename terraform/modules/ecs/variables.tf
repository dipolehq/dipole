variable "tenant_id" {
  description = "Dipole Tenant ID"
  type        = string
}

variable "cloud_endpoint" {
  description = "Dipole Cloud gRPC endpoint for the proxy (e.g., cloud.dipolehq.com:443)"
  type        = string
}

variable "cloud_http_endpoint" {
  description = "Dipole Cloud HTTP endpoint for the controller (defaults to https://<tenant_id>.dipolehq.com)"
  type        = string
  default     = ""
}

variable "device_code" {
  description = "6-digit device code for Agent Controller registration (mutually exclusive with controller_id)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "controller_id" {
  description = "Agent Controller UUID for Trust Provider registration (mutually exclusive with device_code)"
  type        = string
  default     = ""
}

variable "cluster_arn" {
  description = "ARN of the ECS cluster to deploy the Agent Controller into"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID where the ECS services will run"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for the ECS services (private subnets with NAT gateway for Cloud connectivity)"
  type        = list(string)
}

variable "proxy_image" {
  description = "Docker image for the Agent Proxy"
  type        = string
  default     = "dipole/proxy:latest"
}

variable "controller_image" {
  description = "Docker image for the Agent Controller"
  type        = string
  default     = "dipole/controller:latest"
}

variable "namespace" {
  description = "Cloud Map namespace name for service discovery"
  type        = string
  default     = "dipole"
}

variable "controller_cpu" {
  description = "CPU units for the Agent Controller task (1024 = 1 vCPU)"
  type        = number
  default     = 256
}

variable "controller_memory" {
  description = "Memory in MiB for the Agent Controller task"
  type        = number
  default     = 512
}

variable "log_group" {
  description = "CloudWatch Log Group name for container logs"
  type        = string
  default     = "/ecs/dipole"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "ap-south-1"
}

variable "resource_set_id" {
  description = "Dipole Resource Set ID (optional)"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
