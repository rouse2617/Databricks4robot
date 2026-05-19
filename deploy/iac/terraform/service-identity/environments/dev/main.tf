terraform {
  required_version = ">= 1.14.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.29.0"
    }
  }

  # Enable a remote backend before going to prod. Example:
  # backend "gcs" {
  #   bucket = "tf-state-cyber-databrew"
  #   prefix = "envs/dev/service-identity"
  # }
}

provider "google" {
  project = var.project_id
}

# Backend (Go API) workload identity.
module "backend" {
  source = "../../modules/service_identity"

  project_id   = var.project_id
  account_id   = "cyber-databrew-backend"
  display_name = "cyber-databrew backend (dev)"

  # Trim and adjust to least privilege before enabling.
  project_roles = [
    # "roles/secretmanager.secretAccessor",
    # "roles/storage.objectAdmin",
    # "roles/pubsub.publisher",
  ]

  kubernetes_namespace = var.kubernetes_namespace
  kubernetes_sa_name   = "cyber-databrew-backend"
}

# Add additional workloads (frontend, dagster, mcap-tool, ...) below as needed.
