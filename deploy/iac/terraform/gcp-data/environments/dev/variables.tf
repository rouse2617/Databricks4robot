variable "project_id" {
  description = "GCP project id."
  type        = string
}

variable "derived_bucket_name" {
  description = "Name of the derived-artifacts GCS bucket. Empty disables creation."
  type        = string
  default     = ""
}

variable "bucket_location" {
  description = "GCS bucket location."
  type        = string
  default     = "US"
}

variable "bucket_storage_class" {
  description = "GCS bucket default storage class."
  type        = string
  default     = "STANDARD"
}

variable "bucket_soft_delete_retention_seconds" {
  description = "Soft-delete retention seconds."
  type        = number
  default     = 604800
}

variable "bucket_labels" {
  description = "Labels to apply to the derived bucket."
  type        = map(string)
  default     = {}
}

variable "iceberg_buckets" {
  description = "Iceberg-managed buckets for dev (warehouse, datalake)."
  type = map(object({
    location      = string
    storage_class = optional(string, "STANDARD")
    labels        = optional(map(string), {})
  }))
  default = {}
}

variable "pubsub_topics" {
  description = "Pub/Sub topic names to manage."
  type        = list(string)
  default     = []
}
