# Task: Research — How Projects Use/Embed Argo Workflows

Research how other teams and products integrate Argo Workflows into their own UIs and platforms.

## Research Questions

### 1. Argo Workflows UI Patterns
- How do teams at scale (Netflix, Uber, Ant Group, etc.) embed Argo Workflows?
- Is there a common pattern: build custom UI on top of Argo API vs use native Argo vs embed iframe?
- Any open-source reference implementations or starter kits for Argo UI?

### 2. Similar Platforms (Competitive Research)
How do similar workflow/ML platforms handle these scenarios:
- **Kubeflow Pipelines** — How does their UI compare?
- **Apache Airflow** — Rich filtering, log viewing, DAG visualization patterns
- **Prefect / Dagster** — Modern workflow UI patterns
- **MLflow** — Experiment tracking meets pipelines
- **Temporal** — Workflow-as-code UI

### 3. Argo Community Patterns
- Argo Workflows has a separate UI (argo-server) — do people typically use it or replace it?
- Are there popular third-party Argo UIs?
- What do the Argo GitHub issues/discussions say about UI customization?

### 4. Technical Integration Patterns
- **SSE/EventSource for logs:** Standard practice or alternatives?
- **Proxying Argo API:** How do teams handle auth between their app and Argo?
- **Multi-tenant workflow management:** Namespace-per-user vs labels-per-user?

### 5. Ant Design + Workflow UI Examples
- Any open-source projects using Ant Design for workflow/MLOps dashboards?
- How do mature Ant Design apps handle real-time data (EventSource, WebSocket)?

## Deliverable

Write findings to `docs/review/argo-integration-research.md` in the repo at `/Users/rick/cyber-databrew/`. Include:
- Summary of common integration patterns (pros/cons for each)
- 3-5 reference implementations worth studying
- Technical recommendations for our specific stack (React + Ant Design + Go backend)
- Pitfalls to avoid

## Tools
- Web search for market research
- Browse GitHub for open-source Argo UI projects
- Read the Argo docs and community discussions
