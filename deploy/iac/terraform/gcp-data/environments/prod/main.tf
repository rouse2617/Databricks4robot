terraform {
  required_version = ">= 1.14.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.29.0"
    }
  }

  # Org convention mirrors `terraform/prod/<...>/` paths already in use
  # (e.g. `terraform/prod/dagster-code-locations/`). Prod resources for
  # cyber-databrew live in the same GCP project as dev for now, so we keep
  # state in the same `terraform_staging_state_store` bucket but under a
  # `terraform/prod/` prefix.
  backend "gcs" {
    bucket = "terraform_staging_state_store"
    prefix = "terraform/prod/cyber-databrew/gcp-data"
  }
}

provider "google" {
  project = var.project_id
}

module "data" {
  source = "../../modules/gcp_data"

  project_id = var.project_id

  # Iceberg lakehouse (`s3://warehouse/` and `s3://datalake/` in dev/MinIO map here).
  iceberg_buckets = var.iceberg_buckets

  # No derived bucket in prod stack yet; add when needed.
  derived_bucket_name = ""

  pubsub_topics = var.pubsub_topics
}
