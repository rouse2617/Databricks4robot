output "email" {
  description = "Service account email. Use as `iam.gke.io/gcp-service-account` annotation on the KSA."
  value       = google_service_account.this.email
}

output "name" {
  description = "Fully qualified service account name (projects/.../serviceAccounts/...)."
  value       = google_service_account.this.name
}

output "unique_id" {
  description = "Stable numeric id of the service account."
  value       = google_service_account.this.unique_id
}
