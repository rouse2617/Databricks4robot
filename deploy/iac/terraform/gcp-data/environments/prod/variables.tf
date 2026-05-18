variable "project_id" {
  description = "GCP project id."
  type        = string
}

variable "iceberg_buckets" {
  description = "Iceberg-managed buckets for prod (warehouse, datalake)."
  type = map(object({
    location      = string
    storage_class = optional(string, "STANDARD")
    labels        = optional(map(string), {})
  }))
  default = {}
}

variable "pubsub_topics" {
  description = "Pub/Sub topic names to manage in prod."
  type        = list(string)
  default     = []
}
