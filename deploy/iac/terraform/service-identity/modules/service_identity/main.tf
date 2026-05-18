terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
  }
}

# One Google service account per logical workload. Project IAM bindings are
# attached to this SA, and the matching Kubernetes ServiceAccount in
# `var.kubernetes_namespace` is bound via Workload Identity so pods using
# `serviceAccountName: <kubernetes_sa_name>` inherit these GCP roles.
resource "google_service_account" "this" {
  project      = var.project_id
  account_id   = var.account_id
  display_name = var.display_name
  description  = var.description
}

resource "google_project_iam_member" "roles" {
  for_each = toset(var.project_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.this.email}"
}

# Workload Identity binding: KSA in cluster -> this GSA.
resource "google_service_account_iam_member" "workload_identity" {
  count              = var.kubernetes_sa_name == "" ? 0 : 1
  service_account_id = google_service_account.this.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.kubernetes_namespace}/${var.kubernetes_sa_name}]"
}
