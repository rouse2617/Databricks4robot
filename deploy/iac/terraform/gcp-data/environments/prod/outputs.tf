output "iceberg_buckets" {
  description = "Iceberg buckets created by this stack (gs:// urls)."
  value       = module.data.iceberg_buckets
}

output "pubsub_topics" {
  description = "Pub/Sub topic ids."
  value       = module.data.pubsub_topics
}
