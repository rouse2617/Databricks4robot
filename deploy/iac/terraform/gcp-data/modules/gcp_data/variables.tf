variable "project_id" {
  description = "GCP project id."
  type        = string
}

variable "derived_bucket_name" {
  description = "Name of the GCS bucket for derived artifacts. Empty disables creation."
  type        = string
  default     = ""
}

variable "bucket_location" {
  description = "GCS bucket location (region or multi-region) for the derived bucket."
  type        = string
  default     = "US"
}

variable "bucket_storage_class" {
  description = "GCS storage class for the derived bucket."
  type        = string
  default     = "STANDARD"
}

variable "bucket_soft_delete_retention_seconds" {
  description = "Soft-delete retention seconds for the derived bucket."
  type        = number
  default     = 604800
}

variable "bucket_labels" {
  description = "Labels to apply to the derived bucket."
  type        = map(string)
  default     = {}
}

variable "iceberg_buckets" {
  description = <<-EOT
    Iceberg-managed buckets (e.g. warehouse / datalake). Each bucket is created
    with versioning OFF and soft_delete=0s — required defaults for Iceberg
    workloads (Iceberg manages its own snapshotting and frequently deletes small
    files).
    Map key is the bucket name; value carries per-bucket settings.
  EOT
  type = map(object({
    location      = string
    storage_class = optional(string, "STANDARD")
    labels        = optional(map(string), {})
  }))
  default = {}
}

variable "pubsub_topics" {
  description = "List of Pub/Sub topic names to manage."
  type        = list(string)
  default     = []
}
