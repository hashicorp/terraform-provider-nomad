resource "nomad_acl_auth_method" "my_nomad_acl_auth_method" {
  name              = "my-nomad-acl-auth-method"
  type              = "OIDC"
  token_locality    = "global"
  max_token_ttl     = "10m0s"
  token_name_format = "$${auth_method_type}-$${value.user}"
  default           = true

  config {
    oidc_discovery_url    = "https://uk.auth0.com/"
    oidc_client_id        = "someclientid"
    oidc_client_secret    = "someclientsecret-t"
    bound_audiences       = ["someclientid"]
    allowed_redirect_uris = [
      "http://localhost:4649/oidc/callback",
      "http://localhost:4646/ui/settings/tokens",
    ]
    list_claim_mappings = {
      "http://nomad.internal/roles" : "roles"
    }
  }
}
