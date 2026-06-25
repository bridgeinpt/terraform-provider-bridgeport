# Registry connections are imported by their natural key: "environment/name".
# Credentials can't be recovered — re-declare token_wo/password_wo afterward.
terraform import bridgeport_registry_connection.do production/digitalocean
