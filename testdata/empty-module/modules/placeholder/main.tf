# Empty module - no resources, just a placeholder
variable "unused" {
  type    = string
  default = ""
}

output "message" {
  value = "This module has no resources"
}
