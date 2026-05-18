# Lakehouse on GKE — BigQuery + Nessie + MinIO

PoC stack without GCS dependencies. Iceberg table files are stored in MinIO
(`s3://warehouse/bigquery-iceberg`) and metadata is managed by Nessie.

## Apply

```bash
kubectl apply -f deploy/k8s/namespaces.yaml
kubectl apply -n cyber-databrew-dev -k deploy/k8s/lakehouse-minio/
kubectl -n cyber-databrew-dev wait --for=condition=complete job/minio-init --timeout=180s
kubectl -n cyber-databrew-dev rollout status deployment/minio --timeout=180s
kubectl -n cyber-databrew-dev rollout status deployment/nessie --timeout=180s
kubectl -n cyber-databrew-dev rollout status deployment/bigquery --timeout=300s
```

## Smoke

```bash
kubectl -n cyber-databrew-dev exec deployment/bigquery -- bigquery --server http://localhost:8080 --execute "SHOW CATALOGS"
kubectl -n cyber-databrew-dev exec deployment/bigquery -- bigquery --server http://localhost:8080 --execute "CREATE SCHEMA IF NOT EXISTS iceberg.poc"
kubectl -n cyber-databrew-dev exec deployment/bigquery -- bigquery --server http://localhost:8080 --execute "CREATE TABLE IF NOT EXISTS iceberg.poc.hello (x bigint) WITH (format='PARQUET')"
kubectl -n cyber-databrew-dev exec deployment/bigquery -- bigquery --server http://localhost:8080 --execute "INSERT INTO iceberg.poc.hello VALUES (1)"
kubectl -n cyber-databrew-dev exec deployment/bigquery -- bigquery --server http://localhost:8080 --execute "SELECT * FROM iceberg.poc.hello"
```
