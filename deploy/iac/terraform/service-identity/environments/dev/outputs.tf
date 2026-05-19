output "backend_gsa_email" {
  description = "Use as iam.gke.io/gcp-service-account annotation on the backend KSA."
  value       = module.backend.email
}
