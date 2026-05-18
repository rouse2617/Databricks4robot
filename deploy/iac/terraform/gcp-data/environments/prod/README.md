# `gcp-data` / prod

Manages prod data-plane GCP resources.

## Resources currently in scope

- **Iceberg lakehouse buckets** (`warehouse`, `datalake`) — versioning off,
  soft-delete 0s (see module README for rationale).

Add later as prod requirements grow:

- Cloud SQL (separate stack: `gcp-cloud-sql/prod/`)
- Pub/Sub topics (extend `pubsub_topics` here)
- Prod derived bucket (set `derived_bucket_name`)

## First-time apply

```bash
cp terraform.tfvars.example terraform.tfvars   # adjust labels / names if needed

terraform init
terraform plan                # buckets are NEW → expect "+ create" entries
terraform apply
```

Buckets do not exist yet; this is a clean create, not an import.

## State location

```
gs://terraform_staging_state_store/terraform/prod/cyber-databrew/gcp-data/default.tfstate
```

Prod resources still live in `green-valley-442103` (org "staging" project),
so prod state is hosted in the same bucket but under the `terraform/prod/`
prefix — matching the org-wide pattern already used by
`terraform/prod/dagster-code-locations/`.
