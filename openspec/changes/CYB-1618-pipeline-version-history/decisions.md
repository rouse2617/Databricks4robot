# Decisions — CYB-1618

## 2026-06-03 — Research-driven design choices

- **Context**: Researched Prefect, Dagster, Argo Workflows, Jenkins, and Temporal for pipeline version management UX patterns.
- **Key findings**:
  - Prefect auto-versioning with Git metadata + one-click rollback sets the gold standard
  - Dagster collapses "no change" entries in deployment history for readability
  - Argo Workflows uses label-based filtering naturally (fits our K8s-native architecture)
  - Jenkins emphasizes progressive disclosure and resizable panels
- **Decision**: Adopt the timeline+metadata pattern from Prefect/Dagster. Use a drawer (non-blocking). Add a new `GET /api/v1/pipelines/:name/versions` endpoint since the repository already supports `FindVersionsByName`. Keep version filter as a dropdown matching existing execution list patterns.
- **Sources**:
  - https://docs.prefect.io/v3/how-to-guides/deployments/versioning
  - https://docs.dagster.io/guides/build/projects/dagster-plus-project-history
  - https://blog.argoproj.io/whats-new-in-argo-workflows-v3-5-f260e8603ca6
  - https://www.jenkins.io/blog/2025/07/24/redesigning-jenkins-part-two/

## 2026-06-03 — New endpoint needed

- **Context**: Initial plan was frontend-only. Discovered the list API returns only the latest version per template.
- **Decision**: Add `GET /api/v1/pipelines/:name/versions` — the repository already has `FindVersionsByName`. Minimal backend change (one route, one handler).
- **Rationale**: Without this, the frontend can't show version history. The repo method exists but isn't exposed via HTTP.

## 2026-06-03 — Drawer vs Modal

- **Decision**: Right-side Ant Design Drawer.
- **Rationale**: Non-blocking, consistent with app patterns. Prefect uses a tab, Dagster uses a modal — drawer is the best fit for our SPA layout.
