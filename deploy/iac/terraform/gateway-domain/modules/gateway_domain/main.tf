terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
    cloudflare = {
      source = "cloudflare/cloudflare"
    }
  }
}

data "cloudflare_zone" "root" {
  filter = {
    name = var.cloudflare_zone
  }
}

locals {
  frontend_dns_auth_name = "${var.name_prefix}-dns-auth"
  api_dns_auth_name      = "${var.name_prefix}-api-dns-auth"
  frontend_cert_name     = "${var.name_prefix}-cert"
  api_cert_name          = "${var.name_prefix}-api-cert"
  frontend_entry_name    = "${var.name_prefix}-entry"
  api_entry_name         = "${var.name_prefix}-api-entry"
}

resource "google_certificate_manager_dns_authorization" "frontend" {
  name     = local.frontend_dns_auth_name
  domain   = var.frontend_domain
  location = "global"
}

resource "google_certificate_manager_dns_authorization" "api" {
  name     = local.api_dns_auth_name
  domain   = var.api_domain
  location = "global"
}

resource "google_certificate_manager_certificate" "frontend" {
  name     = local.frontend_cert_name
  location = "global"

  managed {
    domains            = [var.frontend_domain]
    dns_authorizations = [google_certificate_manager_dns_authorization.frontend.id]
  }
}

resource "google_certificate_manager_certificate" "api" {
  name     = local.api_cert_name
  location = "global"

  managed {
    domains            = [var.api_domain]
    dns_authorizations = [google_certificate_manager_dns_authorization.api.id]
  }
}

resource "google_certificate_manager_certificate_map_entry" "frontend" {
  name         = local.frontend_entry_name
  map          = var.certificate_map_name
  hostname     = var.frontend_domain
  certificates = [google_certificate_manager_certificate.frontend.id]
}

resource "google_certificate_manager_certificate_map_entry" "api" {
  name         = local.api_entry_name
  map          = var.certificate_map_name
  hostname     = var.api_domain
  certificates = [google_certificate_manager_certificate.api.id]
}

resource "cloudflare_dns_record" "frontend_a" {
  zone_id = data.cloudflare_zone.root.zone_id
  name    = var.frontend_record_name
  type    = "A"
  content = var.gateway_ip
  proxied = true
  ttl     = 1
}

# Single-label subdomain (*.cyberorigin.ai): Orange Cloud works.
# For deeper names like api.<label>.cyberorigin.ai use DNS-only or CF Advanced Certs.
resource "cloudflare_dns_record" "api_a" {
  zone_id = data.cloudflare_zone.root.zone_id
  name    = var.api_record_name
  type    = "A"
  content = var.gateway_ip
  proxied = true
  ttl     = 1
}

resource "cloudflare_dns_record" "frontend_acme_cname" {
  zone_id = data.cloudflare_zone.root.zone_id
  name    = trimsuffix(google_certificate_manager_dns_authorization.frontend.dns_resource_record[0].name, ".${var.cloudflare_zone}.")
  type    = "CNAME"
  content = trimsuffix(google_certificate_manager_dns_authorization.frontend.dns_resource_record[0].data, ".")
  proxied = false
  ttl     = 1
}

resource "cloudflare_dns_record" "api_acme_cname" {
  zone_id = data.cloudflare_zone.root.zone_id
  name    = trimsuffix(google_certificate_manager_dns_authorization.api.dns_resource_record[0].name, ".${var.cloudflare_zone}.")
  type    = "CNAME"
  content = trimsuffix(google_certificate_manager_dns_authorization.api.dns_resource_record[0].data, ".")
  proxied = false
  ttl     = 1
}
