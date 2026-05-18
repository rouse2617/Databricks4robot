variable "project_id" {
  description = "GCP project id."
  type        = string
}

variable "kubernetes_namespace" {
  description = "Kubernetes namespace hosting the workload service accounts."
  type        = string
  default     = "cyber-databrew-dev"
}
