variable "app_name" {
  type = string
}

resource "aws_ecs_cluster" "this" {
  name = var.app_name
}

resource "aws_ecs_service" "this" {
  name            = var.app_name
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.this.arn
  desired_count   = 1
}

resource "aws_ecs_task_definition" "this" {
  family                   = var.app_name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512

  container_definitions = jsonencode([{
    name  = var.app_name
    image = "nginx:latest"
  }])
}

# Nested module call
module "database" {
  source  = "../database"
  db_name = "${var.app_name}-db"
}
