terraform {
  required_version = ">= 1.14.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.29.0"
    }
  }
}

# GCS bucket for derived artifacts written by the backend (`GCS_DERIVED_BUCKET`).
resource "google_storage_bucket" "derived" {
  count                       = var.derived_bucket_name == "" ? 0 : 1
  project                     = var.project_id
  name                        = var.derived_bucket_name
  location                    = var.bucket_location
  storage_class               = var.bucket_storage_class
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false

  soft_delete_policy {
    retention_duration_seconds = var.bucket_soft_delete_retention_seconds
  }

  labels = var.bucket_labels

  lifecycle {
    prevent_destroy = true
  }
}

# Iceberg-managed GCS buckets (warehouse / datalake / staging).
# Differences vs. `derived`:
# - GCS object versioning OFF: Iceberg manages snapshots itself; double-versioning explodes storage cost.
# - Soft-delete 0s: Iceberg compaction and `expireSnapshots` delete many small files; 7-day default
#   would balloon retained-data billing.
# - Same uniform IAM, same enforced public-access, same prevent_destroy as derived.
resource "google_storage_bucket" "iceberg" {
  for_each                    = var.iceberg_buckets
  project                     = var.project_id
  name                        = each.key
  location                    = each.value.location
  storage_class               = lookup(each.value, "storage_class", "STANDARD")
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false

  versioning {
    enabled = false
  }

  soft_delete_policy {
    retention_duration_seconds = 0
  }

  labels = lookup(each.value, "labels", {})

  lifecycle {
    prevent_destroy = true
  }
}

# Pub/Sub topics consumed by the backend / Dagster.
resource "google_pubsub_topic" "topics" {
  for_each = toset(var.pubsub_topics)
  project  = var.project_id
  name     = each.value
}
