output "derived_bucket" {
  description = "GCS bucket for derived artifacts. Empty when not managed here."
  value       = try(google_storage_bucket.derived[0].name, "")
}

output "iceberg_buckets" {
  description = "Map of iceberg bucket short-name to fully qualified GCS url."
  value       = { for k, v in google_storage_bucket.iceberg : k => "gs://${v.name}" }
}

output "pubsub_topics" {
  description = "Map of topic short-name to fully qualified id."
  value       = { for k, v in google_pubsub_topic.topics : k => v.id }
}
