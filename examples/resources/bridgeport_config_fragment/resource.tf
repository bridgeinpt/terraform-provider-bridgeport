# Manage a reusable config fragment.
resource "bridgeport_config_fragment" "common_headers" {
  environment = "production"
  name        = "common-headers"
  content     = <<-EOT
    X-Frame-Options: DENY
    X-Content-Type-Options: nosniff
  EOT
  description = "Shared security headers"
}
