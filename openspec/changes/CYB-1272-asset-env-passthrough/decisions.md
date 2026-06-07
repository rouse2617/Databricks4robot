## 2026-06-07 — OpenSpec checkpoint approved
- **Context**: CYB-1272 changes runtime backend behavior by completing asset context env injection for pipeline runs.
- **Decision**: User approved the OpenSpec checkpoint in chat with "ok".
- **Alternatives**: Continue editing without recording the checkpoint.
- **Rationale**: `docs/agents/AI-RULES.md` requires the OpenSpec checkpoint before runtime code changes.

## 2026-06-07 — Backend dev deploy recovery
- **Context**: Docker credential helper `docker-credential-gcloud` was unavailable locally, and the first Artifact Registry push produced an OCI index where Cloud Run kept pulling a stale digest.
- **Decision**: Used a temporary `DOCKER_CONFIG` with `gcloud auth print-access-token | docker login`, then pushed the SHA tag and `cloudrun-dev-latest` with `docker push --platform linux/amd64`. After Cloud Run created revision `cyber-databrew-backend-dev-00643-qzd`, switched dev traffic to that revision explicitly.
- **Alternatives**: Use Cloud Build, or keep redeploying the multi-platform tag.
- **Rationale**: `docs/agents/deploy-before-commit.md` prefers local Docker build/push for backend dev; forcing linux/amd64 made the deployed digest match the binary containing CYB-1272 changes.

## 2026-06-07 — Deploy smoke route selection
- **Context**: The self-authored tasks originally mentioned a health endpoint smoke. The dev Cloud Run URL returned a platform 404 for both `/health` and `/healthz`, while authenticated core pipeline APIs returned 200.
- **Decision**: Treated `GET /api/v1/pipelines`, `GET /api/v1/assets/CYB10A01`, and the target dry-run manifest check as the deploy acceptance evidence for this backend-only change.
- **Alternatives**: Block CYB-1272 on the unrelated health route behavior.
- **Rationale**: CYB-1272 changes pipeline deployment env assembly, so the dry-run manifest on the deployed revision is the direct acceptance signal.

## 2026-06-07 — Clean SHA deploy for PR regression
- **Context**: The first dev deploy happened before the git commit and used a dirty working-tree tag.
- **Decision**: After committing `04f3ce7`, rebuilt and pushed `cyber-databrew-backend:04f3ce7`, deployed it to backend dev, and reran CYB-1272 API smoke plus Chrome DevTools MCP pipeline regression.
- **Alternatives**: Keep the pre-commit dirty tag as the only deploy evidence.
- **Rationale**: The PR regression evidence should point at the immutable commit SHA that is being merged.
