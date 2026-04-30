ephemeral "nomad_node_intro_token" "client" {
  node_name = "bootstrap-client"
  node_pool = "default"
  ttl       = "15m"
}

# Reference the signed JWT as:
# ephemeral.nomad_node_intro_token.client.jwt
