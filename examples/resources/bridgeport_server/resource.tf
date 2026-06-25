# Manage a server within an existing environment.
resource "bridgeport_server" "web" {
  environment = "production"
  name        = "web-1"
  hostname    = "10.0.0.10"
  public_ip   = "203.0.113.10"
  tags        = ["web", "edge"]
}

output "web_server_id" {
  value = bridgeport_server.web.id
}
