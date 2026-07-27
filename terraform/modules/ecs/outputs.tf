output "controller_service_discovery_endpoint" {
  description = "DNS name for the Agent Controller (use as DIPOLE_AGENT_CONTROLLER in proxy sidecar)"
  value       = "http://agent-controller.${var.namespace}:80"
}

output "controller_security_group_id" {
  description = "Security Group ID of the Agent Controller"
  value       = aws_security_group.controller.id
}

output "proxy_sidecar_container_definition" {
  description = "JSON container definition for the proxy sidecar — merge into your task definition's containerDefinitions"
  value = jsonencode({
    name      = "dipole-agent-proxy"
    image     = var.proxy_image
    essential = false

    environment = concat(
      [
        { name = "DIPOLE_TENANT_ID", value = var.tenant_id },
        { name = "DIPOLE_AGENT_CONTROLLER", value = "http://agent-controller.${var.namespace}:80" },
        { name = "DIPOLE_CLOUD_GRPC_ENDPOINT", value = var.cloud_endpoint },
        { name = "DIPOLE_HTTP_SERVER_PORT", value = "8000" },
        { name = "DIPOLE_METRICS_PORT", value = "9099" },
        { name = "DIPOLE_LOG_LEVEL", value = "info" },
        { name = "DIPOLE_PASS_THROUGH_TRAFFIC_BEFORE_REGISTRATION", value = "true" },
      ],
      var.resource_set_id != "" ? [{ name = "DIPOLE_RESOURCE_SET_ID", value = var.resource_set_id }] : [],
    )

    portMappings = [
      { containerPort = 9099, protocol = "tcp" },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.dipole.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "proxy"
      }
    }
  })
}

output "proxy_environment_variables" {
  description = "Environment variables to add to your app container for explicit proxy steering"
  value = [
    { name = "HTTP_PROXY", value = "http://localhost:8000" },
    { name = "HTTPS_PROXY", value = "http://localhost:8000" },
  ]
}

output "cluster_arn" {
  description = "ECS Cluster ARN (pass-through for reference)"
  value       = var.cluster_arn
}
