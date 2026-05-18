output "derived_bucket" {
  description = "GCS bucket for derived artifacts."
  value       = module.data.derived_bucket
}

output "pubsub_topics" {
  description = "Pub/Sub topic ids managed by this stack."
  value       = module.data.pubsub_topics
}
