# Manage a text config file, optionally composed from fragments.
resource "bridgeport_config_fragment" "common_headers" {
  environment = "production"
  name        = "common-headers"
  content     = "X-Frame-Options: DENY\n"
}

resource "bridgeport_config_file" "nginx" {
  environment  = "production"
  name         = "nginx-conf"
  filename     = "nginx.conf"
  language     = "nginx"
  content      = "server { listen 80; }\n"
  fragment_ids = [bridgeport_config_fragment.common_headers.id]
}
