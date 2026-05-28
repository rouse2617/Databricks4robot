# CYB-1345: Fork Argo UI -- Replace Submit with Canvas

## Background

Analysis recommends forking Argo UI, keeping full detail pages (DAG, logs, params, YAML), replacing only Submit page with our drag-and-drop canvas.

Reference: `docs/review/argo-integration-analysis.md` on `discuss/argo-integration` branch.

## Tasks

### 0. Linear Issue
- Update CYB-1345 from RFE to Implementation status
- Add implementation plan to description
- CC has Linear MCP -- use it directly

### 1. Fork Argo UI

From `/Users/rick/src/argo-workflows/` to `~/cyber-databrew/argo-ui/`

- Copy `ui/` directory (frontend source only)
- Keep structure: `argo-ui/src/`
- Keep webpack build config
- Copy `argo-ui/package.json` and install deps
- Ensure `npm run build` / `npm start` works

### 2. Route Changes

Current routes in `app-router.tsx`:
- /workflows/:namespace? -> list page
- /workflows/:namespace/:name -> detail page (DAG, logs, params, YAML)
- Submit entry in workflow-list sidebar panel

Goal: Replace Submit entry:
- Add route `/workflows/new` or `/submit` for our canvas page
- Keep all other routes unchanged
- Point "Submit New Workflow" button/links to our new page

### 3. Canvas Stub Page

Create `src/app/workflow-submit/`:
- A React page component matching Argo UI style
- Page body: placeholder `<div>Canvas Area</div>` 
- "Submit" button calling Argo's submit API or our compatible API
- Keep styling consistent with Argo UI

### 4. API Adapter Layer

Create `src/shared/adapters/`:
- Wrap Cyber backend API calls:
  - submit: `POST /api/v1/workflows/{namespace}/{name}/submit` -> Cyber pipeline submit
  - list: `GET /api/v1/workflows/{namespace}`
  - detail: `GET /api/v1/workflows/{namespace}/{name}`
- Field mapping: Argo model -> Cyber simplified model
- Real-time: fallback to polling if no SSE

### 5. Auth Integration

- Cyber backend uses `X-Databrew-Token` header
- Inject this header in Argo's `requests.ts` or `base.ts`
- 401 handling: redirect to Cyber login instead of Argo `/login`

### 6. Verification

- `npm start` runs locally, list/detail pages work
- Submit page renders canvas stub
- API calls reach Cyber backend (configurable base URL)

## Key References

- Argo source: `/Users/rick/src/argo-workflows/ui/src/`
- Analysis doc: `git show discuss/argo-integration:docs/review/argo-integration-analysis.md`
- Cyber backend: `/Users/rick/cyber-databrew/backend/`
- Cyber frontend: `/Users/rick/cyber-databrew/Frontend/src/`

## Constraints

- Do NOT modify existing `backend/` or `Frontend/` code -- only `argo-ui/`
- Argo uses React 18 + react-router 4 + webpack, dont migrate tech stack
- Auth = basic header injection, no SSO needed
- Commit all changes to `argo-ui/` with conventional commit format
- Node v22.15.0 available at `/Users/rick/.local/node-v22.15.0/bin/node`
