# `service-identity` / dev

Manages GCP service accounts (GSAs) and Workload Identity bindings for
workloads running in namespace `cyber-databrew-dev`.

## First-time apply

If the GSAs already exist (manually created), import them first; otherwise
skip directly to `terraform apply`.

```bash
cp terraform.tfvars.example terraform.tfvars  # adjust if needed

terraform init
terraform import \
  module.backend.google_service_account.this \
  projects/green-valley-442103/serviceAccounts/cyber-databrew-backend@green-valley-442103.iam.gserviceaccount.com

terraform plan
terraform apply
```

## Wiring K8s side

After `apply`, annotate the matching KSA so pods inherit the GSA:

```bash
kubectl -n cyber-databrew-dev annotate sa cyber-databrew-backend \
  iam.gke.io/gcp-service-account="$(terraform output -raw backend_gsa_email)" \
  --overwrite
```

The KSA itself stays in `deploy/k8s/`. Layer A only owns the **identity**
and its **GCP-side IAM**.
