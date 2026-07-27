# Dipole ECS Fargate Module

Deploys the Dipole Agent Controller as an ECS Fargate service and outputs a proxy sidecar container definition for injection into your task definitions.

## Usage

```hcl
module "dipole" {
  source = "github.com/dipolehq/dipole//terraform/modules/ecs"

  tenant_id      = "your-tenant"
  cloud_endpoint = "your-tenant.dipolehq.com:443"
  device_code    = "123456"
  cluster_arn    = aws_ecs_cluster.main.arn
  vpc_id         = aws_vpc.main.id
  subnet_ids     = aws_subnet.private[*].id
  region         = "ap-south-1"
}

resource "aws_ecs_task_definition" "my_app" {
  family                   = "my-app"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024

  container_definitions = jsonencode([
    {
      name      = "app"
      image     = "my-app:latest"
      essential = true
      environment = concat(
        [{ name = "APP_PORT", value = "3000" }],
        jsondecode(module.dipole.proxy_environment_variables),
      )
    },
    jsondecode(module.dipole.proxy_sidecar_container_definition),
  ])
}
```

## Requirements

- ECS cluster must be in a VPC with NAT Gateway (proxy needs outbound HTTPS to Cloud)
- Private subnets for both Controller and app tasks
- Service discovery namespace created by this module (uses AWS Cloud Map)

## Inputs

| Name | Description | Type | Default |
|------|-------------|------|---------|
| tenant_id | Dipole Tenant ID | string | (required) |
| cloud_endpoint | Cloud gRPC endpoint | string | (required) |
| cluster_arn | ECS Cluster ARN | string | (required) |
| vpc_id | VPC ID | string | (required) |
| subnet_ids | Private subnet IDs | list(string) | (required) |
| device_code | 6-digit device code for registration | string | "" |
| controller_id | Controller UUID for TP registration | string | "" |
| region | AWS region | string | "ap-south-1" |

## Outputs

| Name | Description |
|------|-------------|
| controller_service_discovery_endpoint | DNS URL for the Controller |
| proxy_sidecar_container_definition | JSON to merge into your task definition |
| proxy_environment_variables | HTTP_PROXY/HTTPS_PROXY env vars for your app container |
