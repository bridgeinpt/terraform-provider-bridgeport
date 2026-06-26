# Secrets are imported by their natural key: "environment/key". The value cannot
# be recovered, so after importing add value_wo and value_wo_version to config.
terraform import bridgeport_secret.db_password production/DB_PASSWORD
