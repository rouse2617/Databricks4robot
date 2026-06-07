# Tasks — CYB-1646

## Context Files
- `Frontend/src/pages/WorkflowDetailPage.tsx` — workflow execution summary, cost rendering, run dialog, and log viewer wiring.
- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx` — node detail monitoring, cost, logs, and troubleshooting copy.
- `Frontend/src/pages/PipelinePage.tsx` — pipeline list run entry point and dialog wiring.
- `Frontend/src/components/pipeline/DeployPanel.tsx` — deploy/run dialog wording.
- `Frontend/src/pages/useWorkflowDetail.test.tsx` — workflow log and detail fixtures.
- `Frontend/src/pages/WorkflowDetailPage.test.tsx` — execution detail rendering coverage.
- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.test.tsx` — node detail empty-state coverage.
- `docs/agents/deploy-verification.md` — frontend deploy and Chrome DevTools MCP verification requirements.

## Implementation
- [x] [Frontend] Replace generic missing-cost labels with runtime-aware labels for pending and completed workflow runs.
- [x] [Frontend] Update node detail monitoring and billing empty states so pending, completed-without-snapshot, and unavailable states have distinct copy.
- [x] [Frontend] Map unavailable or pending log viewer states to Chinese product copy and remove raw English technical explanations from the UI.
- [x] [Frontend] Update pipeline run entry points so no-asset runs use "运行" with a secondary "本次不绑定资产" explanation.
- [x] [Frontend] Clarify run dialog version selection copy without changing workflow version semantics.
- [x] [Frontend] Add or update focused tests for the changed empty states and run dialog copy.

## API Contract Sync
No HTTP API is added or changed in this change. Existing workflow, node, log, and cost APIs remain unchanged.

## Local Verification
- [x] [Frontend] Run focused tests for workflow detail, node detail panel, pipeline page, and deploy panel.
- [x] [Frontend] `npm run lint`
- [x] [Frontend] `npm run build`

### Local Verification Evidence — 2026-06-07

- Focused tests passed: `npm run test -- --run src/pages/WorkflowDetailPage.test.tsx src/components/pipeline/WorkflowNodeDetailPanel.test.tsx src/pages/PipelinePage.test.tsx src/components/pipeline/DeployPanel.test.tsx` (`4` files, `72` tests).
- Frontend lint passed: `npm run lint` (Biome checked `237` files).
- Frontend build passed: `npm run build` (Vite emitted existing large chunk warnings only).

## Deploy Verification
- [x] Deploy frontend dev Worker with `wrangler deploy --env dev`.
- [x] Chrome DevTools MCP on dev: workflow execution detail with completed complex DAG.
- [x] Chrome DevTools MCP on dev: pipeline run dialog/no-asset run entry point.
- [x] Chrome DevTools MCP console/network check has no new runtime errors.
- [x] Record deployed Worker version and tested URLs in this file before commit.

### Deploy Verification Evidence — 2026-06-07

- Deployed Worker: `cyber-databrew-dev`
- Worker URL: `https://cyber-databrew-dev.cyberorigin.ai/`
- Worker deployment: `fad8011d-9d53-4428-a505-37a8761373b8`
- Worker version: `13f8f2aa-0b9c-422a-9164-10047609939a` at `100%`
- Entry bundle check: `/assets/index-CafAI_87.js` contains `environment:w("dev")`.
- Chrome DevTools MCP verified workflow detail:
  `https://cyber-databrew-dev.cyberorigin.ai/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`
  - Footer marker displayed `v0.1.1 · 0da51f3-dirty · dev`.
  - Summary cost, node detail total, and node rows displayed `成本快照未生成`.
  - Asset summary displayed `无资产`.
  - Log drawer displayed Chinese product copy (`尾部`, `上限`, `当前是实时日志窗口`, `暂无日志输出`) instead of raw `tail` / `follow` wording.
  - Network: `auth/me`, `pipeline-runs`, `workflows`, `events`, `asset-nodes`, `cost-summary`, and `logs` requests returned `200`.
- Chrome DevTools MCP verified pipeline run dialog:
  `https://cyber-databrew-dev.cyberorigin.ai/pipeline?tab=pipelines`
  - Saved pipeline action button displayed `运行`.
  - No-asset modal displayed `本次不绑定资产` and primary action `运行`.
  - Version field displayed `默认使用活跃版本；也可以在这里选择历史版本运行。`
- Chrome DevTools MCP verified URL asset context:
  `https://cyber-databrew-dev.cyberorigin.ai/pipeline?tab=pipelines&asset_ids=ast-a,ast-b`
  - Pipeline list displayed `已选择 2 个资产`.
  - Run modal displayed `将处理 2 个资产` and primary action `运行资产`.
- Chrome DevTools MCP fixed page regression for style-scope risk: `4/11` pages passed (`/dashboard`, `/assets`, `/assets/SDKT0202`, `/registry`); each loaded core content with no blank screen.
- Console check: no `error` or `warn` messages found during workflow detail, pipeline dialog, and fixed page regression verification.

## PR
- [ ] PR template filled; Linear CYB-1646 and OpenSpec change ID linked.
