terraform {
  required_providers {
    legocharm = {
      source  = "alexdlukens/legocharm"
      version = "~> 0.0.3"
    }
  }
}

provider "legocharm" {
  address  = var.legocharm_address
  username = var.legocharm_username
  password = var.legocharm_password
}

resource "legocharm_user" "example" {
  username = var.username
  password = var.password
  email    = var.email
}

resource "legocharm_user_domain_access" "example_site_access" {
  user_id = legocharm_user.example.id
  domain  = "example.com"
  access_level = "domain"
}