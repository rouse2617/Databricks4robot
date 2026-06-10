# cyber-databrew Repository Wiki

Generated on: 2026-06-02

A detailed, English repository wiki following the Qoder repo-wiki layout: every page
has a `<cite>` "Referenced Files" block, a table of contents, the standard section
skeleton (Introduction → Project Structure → Core Components → Architecture Overview →
Detailed Component Analysis → Dependency Analysis → Performance Considerations →
Troubleshooting Guide → Conclusion → Appendices), mermaid diagrams, and `Section
sources` / `Diagram sources` citations with real line ranges.

**Markdown is the single source of truth.** The catalog tree, nav order, and the code
each page is derived from are declared in `docs/repo-wiki/manifest.yaml`.

## Hosting on GitHub Pages

The hosted wiki is built by `.github/workflows/repo-wiki-pages.yml`.

On pushes to `main` that touch `docs/repo-wiki/`, `docs/agents/skills/repo-wiki/`,
`scripts/repo-wiki/`, or the workflow itself, GitHub Actions:

1. installs the MkDocs dependencies from `scripts/repo-wiki/requirements-mkdocs.txt`;
2. regenerates `mkdocs.yml` from `manifest.yaml`;
3. builds `docs/repo-wiki-site/` with `mkdocs build --strict`;
4. publishes that generated site with GitHub Pages.

Repository maintainers must configure **Settings → Pages → Build and deployment →
Source → GitHub Actions** before the first deployment.

## Reviewing as HTML

**MkDocs Material** (search, dark mode, collapsible nav, live mermaid):

```bash
pip install -r scripts/repo-wiki/requirements-mkdocs.txt
python3 scripts/repo-wiki/gen_mkdocs.py             # manifest -> mkdocs.yml (regenerate after manifest edits)
mkdocs serve                                        # http://localhost:8000
mkdocs build                                        # -> docs/repo-wiki-site/ (gitignored)
```

The hosted site is published automatically to GitHub Pages — see [Hosting on GitHub Pages](#hosting-on-github-pages).

## Authoring

- `python3 scripts/repo-wiki/scaffold.py` — create any missing page from the manifest (never overwrites).
- Workflows (bootstrap / update / preview) are encoded in `docs/agents/skills/repo-wiki/SKILL.md`.

## Catalog

- [Project Overview](./overview.md)
- [Quick Start](./quick-start.md)

**Architecture** — [System Architecture Overview](./architecture/system-architecture-overview.md) ·
[Modular Design](./architecture/modular-design.md) ·
[Event & CDC Architecture](./architecture/event-cdc-architecture.md) ·
[Search & Query Architecture](./architecture/search-query-architecture.md) ·
[Lakehouse Architecture](./architecture/lakehouse-architecture.md) ·
[Deployment Architecture](./architecture/deployment-architecture.md) ·
[Security Architecture](./architecture/security-architecture.md)

**Asset Management** — [Module](./modules/asset-management/index.md) ·
[Lifecycle](./modules/asset-management/asset-lifecycle.md) ·
[Types & Schema](./modules/asset-management/asset-types-and-schema.md) ·
[Tags & Lineage](./modules/asset-management/tags-and-lineage.md) ·
[Events & Timeline](./modules/asset-management/events-and-timeline.md)

**MCAP & Storage** — [Module](./modules/mcap-storage/index.md) ·
[Files & Messages](./modules/mcap-storage/mcap-files-and-messages.md) ·
[Byte Serving & Locators](./modules/mcap-storage/byte-serving-and-locators.md) ·
[Preview Service](./modules/mcap-storage/preview-service.md)

**Delivery** — [Module](./modules/delivery/index.md) ·
[Commit & Items](./modules/delivery/delivery-commit-and-items.md) ·
[Rules & Eligibility](./modules/delivery/delivery-rules-and-eligibility.md)

**Algorithm Lifecycle** — [Module](./modules/algorithm-lifecycle/index.md) ·
[State Machine](./modules/algorithm-lifecycle/algo-state-machine.md) ·
[Algorithm Runs](./modules/algorithm-lifecycle/algo-runs.md)

**Search & Query** — [Module](./modules/search-query/index.md) ·
[Query IR](./modules/search-query/query-ir.md) ·
[Plan & Execution](./modules/search-query/query-plan-and-exec.md) ·
[Saved Queries](./modules/search-query/saved-queries.md) ·
[Sync & Reindex](./modules/search-query/sync-and-reindex.md)

**Lakehouse** — [Module](./modules/lakehouse/index.md) ·
[BigQuery & BigLake](./modules/lakehouse/bigquery-biglake.md) ·
[Incremental Ingestion](./modules/lakehouse/incremental-ingestion.md)

**Pipeline & Workflows** — [Module](./modules/pipeline-workflows/index.md) ·
[Templates](./modules/pipeline-workflows/pipeline-templates.md) ·
[Deployments](./modules/pipeline-workflows/deployments.md) ·
[Argo Integration](./modules/pipeline-workflows/argo-integration.md) ·
[Components](./modules/pipeline-workflows/pipeline-components.md)

**Registry** — [Module](./modules/registry/index.md) ·
[Registry Types](./modules/registry/registry-types.md)

**Event Processing & CDC** — [Module](./modules/event-cdc/index.md) ·
[Outbox Pattern](./modules/event-cdc/outbox-pattern.md) ·
[Event Schemas](./modules/event-cdc/event-schemas.md) ·
[ES Subscriber](./modules/event-cdc/es-subscriber.md) ·
[Pub/Sub & Kafka](./modules/event-cdc/pubsub-and-kafka-subscribers.md) ·
[Search Indexing](./modules/event-cdc/search-indexing.md)

**Database Design** — [Overview](./data-model/index.md)
&nbsp;·&nbsp; _Data Model Design:_ [Index](./data-model/model-design.md) ·
[Asset](./data-model/asset-model.md) ·
[MCAP & Segment](./data-model/mcap-model.md) ·
[Action](./data-model/action-model.md) ·
[Algorithm Run](./data-model/algo-run-model.md) ·
[Delivery](./data-model/delivery-model.md) ·
[Pipeline](./data-model/pipeline-model.md) ·
[Backfill & Checkpoint](./data-model/backfill-checkpoint-model.md) ·
[Saved Query & Customer](./data-model/saved-query-customer-model.md)
&nbsp;·&nbsp; [Data Integrity & Constraints](./data-model/integrity-and-constraints.md) ·
[Performance & Indexing](./data-model/performance-and-indexing.md) ·
[Migration Management](./data-model/migrations.md)

**Backend API Reference** — [Overview](./api/index.md) ·
[Authentication](./api/authentication.md) ·
[Assets](./api/assets-api.md) ·
[Actions & Eval](./api/actions-eval-api.md) ·
[MCAP](./api/mcap-api.md) ·
[Deliveries](./api/deliveries-api.md) ·
[Algorithm](./api/algo-api.md) ·
[Query Workbench](./api/query-workbench-api.md) ·
[Search & Admin](./api/search-admin-api.md) ·
[Lakehouse](./api/lakehouse-api.md) ·
[Pipeline & Workflow](./api/pipeline-workflow-api.md) ·
[Registry](./api/registry-api.md) ·
[Infrastructure & Audit](./api/infrastructure-api.md)

**Frontend** — [Design](./frontend/index.md) ·
[Layout & Routing](./frontend/layout-and-routing.md) ·
[API Layer](./frontend/api-layer.md) ·
[Hooks System](./frontend/hooks-system.md) ·
[Dashboard](./frontend/dashboard-page.md) ·
[Asset Pages](./frontend/asset-pages.md) ·
[MCAP Page](./frontend/mcap-page.md) ·
[Algorithm Pages](./frontend/algo-pages.md) ·
[Delivery Pages](./frontend/delivery-pages.md) ·
[Registry & Tags](./frontend/registry-tag-pages.md) ·
[Metrics & Events](./frontend/metrics-events-pages.md) ·
[Pipeline & Workflow](./frontend/pipeline-workflow-pages.md) ·
[Shared UI](./frontend/shared-ui-components.md)

**SDK** — [Guide](./sdk/index.md) ·
[Client & Requestor](./sdk/client-and-requestor.md) ·
[Managers](./sdk/managers.md) ·
[Configuration Resolution](./sdk/config-resolution.md)

**Security** — [Overview](./security/index.md) ·
[Authentication Mechanism](./security/authentication-mechanism.md) ·
[Authorization & Roles](./security/authorization-and-roles.md) ·
[Middleware](./security/middleware.md) ·
[Data Protection](./security/data-protection.md)

**Testing** — [Strategy](./testing/index.md) ·
[Backend Tests](./testing/backend-tests.md) ·
[Frontend Tests](./testing/frontend-tests.md) ·
[E2E & Smoke](./testing/e2e-and-smoke.md)

**Operations** — [Deployment & Operations](./operations/index.md) ·
[Local Compose](./operations/local-compose.md) ·
[Kubernetes](./operations/kubernetes.md) ·
[Cloud Run](./operations/cloud-run.md) ·
[Terraform](./operations/terraform.md) ·
[Tekton CI/CD](./operations/tekton-cicd.md) ·
[Migrations Runbook](./operations/migrations-runbook.md) ·
[Monitoring](./operations/monitoring.md)

**Development** — [Tooling](./development/index.md) ·
[Makefile & Scripts](./development/makefile-and-scripts.md) ·
[Agent Workflow & OpenSpec](./development/agent-workflow-and-openspec.md) ·
[Contributing Guide](./development/contributing.md)

- [Troubleshooting](./troubleshooting.md)
