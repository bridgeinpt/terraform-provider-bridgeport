# Manage a non-secret environment variable.
resource "bridgeport_var" "log_level" {
  environment = "production"
  key         = "LOG_LEVEL"
  value       = "info"
  description = "Application log verbosity"
}
