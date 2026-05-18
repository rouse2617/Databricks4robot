terraform {
  required_version = ">= 1.14.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.29.0"
    }
  }

  # Org convention: dev maps to the staging tf state bucket.
  backend "gcs" {
    bucket = "terraform_staging_state_store"
    prefix = "terraform/staging/cyber-databrew/gcp-data"
  }
}

provider "google" {
  project = var.project_id
}

module "data" {
  source = "../../modules/gcp_data"

  project_id                           = var.project_id
  derived_bucket_name                  = var.derived_bucket_name
  bucket_location                      = var.bucket_location
  bucket_storage_class                 = var.bucket_storage_class
  bucket_soft_delete_retention_seconds = var.bucket_soft_delete_retention_seconds
  bucket_labels                        = var.bucket_labels
  iceberg_buckets                      = var.iceberg_buckets
  pubsub_topics                        = var.pubsub_topics
}
