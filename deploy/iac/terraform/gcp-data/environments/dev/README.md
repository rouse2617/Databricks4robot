# `gcp-data` / dev

Manages data-plane GCP resources whose lifecycle is independent of the
backend release cycle: GCS buckets and Pub/Sub topics consumed by the
backend (`GCS_DERIVED_BUCKET`, `TOPIC_*`).

## First-time apply

```bash
cp terraform.tfvars.example terraform.tfvars  # fill in values

terraform init
# Import any existing buckets/topics first, then:
terraform plan
terraform apply
```

## Importing existing resources

```bash
# GCS bucket
terraform import \
  'module.data.google_storage_bucket.derived[0]' \
  cyber-databrew-derived-dev

# Pub/Sub topic (must be in tfvars first)
terraform import \
  'module.data.google_pubsub_topic.topics["mcap-finalized-dev"]' \
  projects/green-valley-442103/topics/mcap-finalized-dev
```

## Notes

- `prevent_destroy = true` on the bucket: removing it from tfvars / config
  will fail `apply` until `lifecycle` is loosened. This is intentional.
- IAM grants on these resources should live in `service-identity/` (per
  GSA), not here. Keep "what exists" separate from "who can use it".
- **Legacy Debezium CDC topics** named `cyber_databrew_dev.public.<table>`
  (and their `cdc-*-sub` subscriptions) are **not** created by this stack’s
  empty `pubsub_topics` list; if they still existed from older experiments,
  delete them in GCP when unused so they are not mistaken for the current
  outbox path (`TOPIC_ASSET_EVENTS` → e.g. `cyber-databrew-asset-events`).
