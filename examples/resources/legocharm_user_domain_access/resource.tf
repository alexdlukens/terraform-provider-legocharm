resource "legocharm_user_domain_access" "example_access" {
  user_id      = legocharm_user.example_user.id
  domain       = "staging.example.com"
  access_level = "subdomain"
}
