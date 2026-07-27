terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

resource "aws_cloudwatch_log_group" "dipole" {
  name              = var.log_group
  retention_in_days = 14
  tags              = var.tags
}

resource "aws_service_discovery_private_dns_namespace" "dipole" {
  name = var.namespace
  vpc  = var.vpc_id
  tags = var.tags
}

resource "aws_service_discovery_service" "controller" {
  name = "agent-controller"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.dipole.id

    dns_records {
      ttl  = 10
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }

  tags = var.tags
}

resource "aws_security_group" "controller" {
  name        = "dipole-agent-controller"
  description = "Agent Controller — allows inbound HTTP from VPC for proxy token requests"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.selected.cidr_block]
    description = "HTTP from VPC (proxy token requests)"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound (Cloud registration + health)"
  }

  tags = var.tags
}

data "aws_vpc" "selected" {
  id = var.vpc_id
}

resource "aws_iam_role" "task_execution" {
  name = "dipole-ecs-task-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "task_execution" {
  role       = aws_iam_role.task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_ecs_task_definition" "controller" {
  family                   = "dipole-agent-controller"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.controller_cpu
  memory                   = var.controller_memory
  execution_role_arn       = aws_iam_role.task_execution.arn

  container_definitions = jsonencode([{
    name      = "controller"
    image     = var.controller_image
    essential = true

    portMappings = [{
      containerPort = 80
      protocol      = "tcp"
    }]

    environment = concat(
      [
        { name = "DIPOLE_TENANT_ID", value = var.tenant_id },
        { name = "DIPOLE_CLOUD_ENDPOINT", value = var.cloud_http_endpoint != "" ? var.cloud_http_endpoint : "https://${var.tenant_id}.dipolehq.com" },
      ],
      var.device_code != "" ? [{ name = "DIPOLE_DEVICE_CODE", value = var.device_code }] : [],
      var.controller_id != "" ? [{ name = "DIPOLE_AGENT_CONTROLLER_ID", value = var.controller_id }] : [],
    )

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.dipole.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "controller"
      }
    }
  }])

  tags = var.tags
}

resource "aws_ecs_service" "controller" {
  name            = "dipole-agent-controller"
  cluster         = var.cluster_arn
  task_definition = aws_ecs_task_definition.controller.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [aws_security_group.controller.id]
    assign_public_ip = false
  }

  service_registries {
    registry_arn = aws_service_discovery_service.controller.arn
  }

  tags = var.tags
}
