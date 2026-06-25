# Config files are imported by their natural key: "environment/name".
# fragment_ids cannot be recovered on import — re-declare them in config.
terraform import bridgeport_config_file.nginx production/nginx-conf
