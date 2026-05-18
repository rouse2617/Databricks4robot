terraform {
  required_version = ">= 1.14.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.29.0"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      # Must stay on v5: module uses `cloudflare_dns_record`; lock file pins 5.x.
      version = "~> 5.0"
    }
  }

  # Org convention: dev maps to the staging tf state bucket.
  backend "gcs" {
    bucket = "terraform_staging_state_store"
    prefix = "terraform/staging/cyber-databrew/gateway-domain"
  }
}

provider "google" {
  project = var.project_id
}

# IaC-style secret injection with fallback:
# - prefer explicit TF_VAR_cloudflare_api_token when provided
# - otherwise read from Secret Manager
data "google_secret_manager_secret_version" "cf_token" {
  count   = var.cloudflare_api_token == "" ? 1 : 0
  project = var.project_id
  secret  = var.cloudflare_api_token_secret_name
  version = "latest"
}

locals {
  effective_cloudflare_api_token = var.cloudflare_api_token != "" ? var.cloudflare_api_token : try(data.google_secret_manager_secret_version.cf_token[0].secret_data, "")
}

provider "cloudflare" {
  api_token = local.effective_cloudflare_api_token
}

module "gateway_domain" {
  source = "../../modules/gateway_domain"

  cloudflare_zone      = var.cloudflare_zone
  certificate_map_name = var.certificate_map_name
  gateway_ip           = var.gateway_ip
  frontend_domain      = var.frontend_domain
  api_domain           = var.api_domain
  frontend_record_name = var.frontend_record_name
  api_record_name      = var.api_record_name
  name_prefix          = var.name_prefix
}
