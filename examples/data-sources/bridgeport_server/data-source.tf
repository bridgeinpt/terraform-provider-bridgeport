# Look up a single server by its natural key (environment + name).
data "bridgeport_server" "web" {
  environment = "production"
  name        = "web-1"
}

output "web_server_private_ip" {
  value = data.bridgeport_server.web.private_ip
}
