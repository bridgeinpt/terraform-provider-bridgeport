# Manage a secret. The value is write-only — it never lands in Terraform state.
# Source it from a write-only-capable source (e.g. an ephemeral value or a
# variable) and bump value_wo_version whenever you change value_wo to rotate it.
variable "db_password" {
  type      = string
  sensitive = true
}

resource "bridgeport_secret" "db_password" {
  environment      = "production"
  key              = "DB_PASSWORD"
  value_wo         = var.db_password
  value_wo_version = "1"
  description      = "Primary database password"
}
