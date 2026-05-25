# cyber-databrew-sdk

Python SDK for the [cyber-databrew](https://github.com/CyberOrigin2077/cyber-databrew) data platform.

## Install

```bash
# 一次性配置（有 gcloud 认证即可）：
pip install keyrings.google-artifactregistry-auth

# 安装 SDK：
pip install --extra-index-url \
  https://us-central1-python.pkg.dev/green-valley-442103/python-packages/simple/ \
  cyber-databrew-sdk
```

## Quick Start

```python
from cyber_databrew_sdk import CyberDatabrewClient

client = CyberDatabrewClient(
    base_url="https://your-backend-url.run.app",
    token="g-your-token",
)

# List assets
assets = client.assets.list(page_size=20)

# Get asset detail
asset = client.assets.get("asset-id")

# Create delivery
delivery = client.deliveries.create(
    customer_id="cust-1",
    items=[{"asset_id": "asset-1", "mcap_file_id": "file-1"}],
)
```

## Config Priority

`CyberDatabrewClient` resolves configuration from multiple sources (highest priority first):

1. Constructor args: `CyberDatabrewClient(base_url=..., token=...)`
2. Environment vars: `CYBER_DATABREW_BASE_URL`, `CYBER_DATABREW_TOKEN`, `CYBER_DATABREW_EMAIL`
3. Config file: `~/.cyber-databrew/config.yaml`
4. Remote fetch: `GET /api/v1/sdk-config` (backend endpoint)
5. Built-in defaults

## Managers

| Manager | Description |
|---------|------------|
| `client.assets` | Asset CRUD, tags, lineage, provenance, timeline |
| `client.storage` | MCAP file list, download, upload |
| `client.delivery` | Delivery create, commit, cancel, ack |
| `client.algo_runs` | Algorithm run lifecycle |
| `client.search` | Sync status / progress |
| `client.queries` | Query IR validate, run, saved queries |
| `client.customers` | Customer CRUD |
| `client.lakehouse` | Lakehouse report, tables, overview |
| `client.events` | Asset / global event list, SSE stream |
| `client.registry` | Algo, tag, metric, lifecycle registries |
| `client.audit` | Audit search, lineage search |

## Development

```bash
cd sdk
uv sync --dev
make validate    # lint → mypy → test → build → smoke
make publish     # publish to Artifact Registry
```

## License

Internal — CyberOrigin
