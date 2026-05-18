variable "project_id" {
  description = "GCP project id."
  type        = string
}

variable "account_id" {
  description = "Service account id (left side of the email)."
  type        = string
}

variable "display_name" {
  description = "Service account display name."
  type        = string
  default     = ""
}

variable "description" {
  description = "Service account description."
  type        = string
  default     = ""
}

variable "project_roles" {
  description = "Project-level IAM roles to grant to this service account."
  type        = list(string)
  default     = []
}

variable "kubernetes_namespace" {
  description = "Kubernetes namespace hosting the bound KSA."
  type        = string
  default     = ""
}

variable "kubernetes_sa_name" {
  description = "Kubernetes ServiceAccount name to bind via Workload Identity. Empty disables the binding."
  type        = string
  default     = ""
}
