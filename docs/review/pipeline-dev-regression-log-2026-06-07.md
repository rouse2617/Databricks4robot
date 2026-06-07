# Pipeline Dev Regression Log — 2026-06-07

Environment:
- Frontend dev: `https://cyber-databrew-dev.cyberorigin.ai`
- Method: Chrome DevTools MCP only
- Scope: pipeline E2E and regression on current `dev`

## Verified working

- Execution list page loads and renders real data.
- Searching execution name `test-77705a` narrows the list to the target run.
- Clicking `查看` opens:
  - `/pipeline/executions/test-77705a?runId=4ec0af22-d1eb-4880-9a62-088a8a5951b0`
- Success execution detail renders:
  - pipeline name `test`
  - version `v2`
  - node count `2`
  - event count `19`
  - business node label `Count Lines` in DAG, timeline, node table, and node detail drawer
- Timeline view can be switched on and renders the two `Count Lines` step bars.
- Failed execution entry `pipeline-20260605-190155-5583db` opens from `查看失败详情`.
- Failed execution detail renders:
  - run status `Failed`
  - failed node summary
  - failed node label `Pass Through`
  - node detail table and event timeline
- Pipeline design page loads:
  - component palette
  - component search
  - deploy/save/export/import toolbar
- Component search filters `Count` to `Count Lines`.
- Component manager page loads real component rows.
- Editing `Count Lines` opens the component form with:
  - command
  - args
  - CPU / memory / disk / GPU
  - compute tier
  - tolerations
  - input/output ports
- Saved pipelines page loads real template rows with:
  - version labels
  - version counts
  - active version marker, e.g. `活跃: v1`
- Saved pipelines `运行` button is safe at first step:
  - clicking `test -> 运行` opens a modal instead of directly creating a new run
  - modal defaults to `版本 v2`
  - modal shows `Default Argo target · default/cyber-databrew-dev`
  - modal warns current mode is `no-asset run`
- Saved pipelines `编辑` works when it navigates with the full UUID:
  - `test` opens correctly at
    - `/pipeline?templateId=c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - the canvas rehydrates with pipeline name `test`, version selector `v2`, and two `Count Lines` nodes
- Multi-version pipeline edit also works with full UUID:
  - `pipeline-20260605-174332` opens correctly at
    - `/pipeline?templateId=dc34200f-e580-492f-99aa-88827e8d7c49`
  - canvas rehydrates with version selector `v3`
  - nodes visible: `find-toni-stats` and `Count Lines`

## Confirmed issues

### 1. Saved pipeline edit flow is broken

Symptoms:
- From saved pipelines, template `test` is shown with visible short ID `c00f1136`.
- Direct navigation using the visible short ID:
  - `/pipeline?templateId=c00f1136`
  - does not load the saved pipeline into the canvas
  - canvas stays in empty-state / new-pipeline mode
- Network evidence:
  - `GET /api/v1/pipelines/c00f1136` -> `404`
  - `GET /api/v1/pipelines/c00f1136/versions` -> `200`
- Console records a resource `404`.

Important detail:
- The saved pipeline list API returns the real full UUID for `test`:
  - `c00f1136-1f86-4a32-a25d-91bf81557fc8`
- But even direct navigation with the full UUID also fails to rehydrate the canvas:
  - `/pipeline?templateId=c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - page still stays in new-pipeline empty-state
- This means the regression is worse than only “short ID vs full UUID”.
- The template detail load path is not populating the canvas in either case.

Reason this is unreasonable:
- The saved pipelines list clearly exposes a pipeline entry and version metadata.
- An edit/deep-link flow that cannot re-open that same template makes saved pipeline management incomplete.

Additional evidence:
- `运行` on the same `test` row does open a valid run modal and correctly resolves `v2`.
- So the saved-pipeline row itself is not generally broken.
- The failure appears specific to the template rehydration / edit path.

Revision to earlier suspicion:
- The edit path is **not** universally broken.
- It works correctly when the route uses the full UUID template id.
- The real problem is narrower:
  - user-visible short IDs are misleading for deep links
  - any code path that constructs `templateId` from the short display id will fail

Systematic API pattern confirmed:
- For multiple pipeline templates, the API behaves like this:
  - `GET /api/v1/pipelines/<full-uuid>` -> `200`
  - `GET /api/v1/pipelines/<short-id>` -> `404`
  - `GET /api/v1/pipelines/<full-uuid>/versions` -> `200`
  - `GET /api/v1/pipelines/<short-id>/versions` -> `200`
- Verified examples include:
  - `test`
  - `34545`
  - `pipeline-20260605-190155`
  - `mcp-complex-fanout-chain`
  - `mcp-complex-fanout-chain-ok`

Why this is especially unreasonable:
- The API contract is inconsistent inside one resource family.
- A caller can incorrectly conclude that short IDs are supported everywhere because `.../versions` succeeds.

### 2. Pod diagnostics API returns forbidden on workflow detail

Route:
- `GET /api/v1/workflows/test-77705a/nodes/test-77705a-4151996238/pod`

Observed result:
- `403`
- body:
  - `{"code":"K8S_FORBIDDEN","message":"Kubernetes credentials cannot read pod diagnostics", ...}`

UI impact:
- Page degrades to empty/fallback runtime copy.
- Console records a failed resource load.

Reason this is unreasonable:
- The node detail UI exposes a `Pod` action and runtime diagnostics panel.
- In dev, the page is partially functional but the diagnostic surface is effectively blocked by backend credentials.

### 3. Asset detail action timeline request returns 404

Route:
- `GET /api/v1/assets/CYB10A01/action-annotations?limit=200`

Observed result:
- `404`

UI impact:
- Console records a failed resource load.
- Related tabs can still settle into empty states, but the route itself is missing.

Reason this is unreasonable:
- The asset detail UI exposes an `Action 时间轴` tab.
- A user-visible tab backed by a missing endpoint is a contract mismatch.

### 4. Asset detail -> file tab has loading delay / unstable completion

Observed:
- `file 文件` tab first shows `加载中...`
- It later settles in some flows, but the transition is inconsistent and slow enough to cause ambiguous verification states.

Reason this is worth tracking:
- It may not be a hard functional failure.
- But it is unstable enough that it can hide real routing or API issues during regression checks.

### 5. Pipeline edit path likely ignores available list payload and depends on failing detail fetch

Observed:
- `GET /api/v1/pipelines` already includes embedded pipeline structure for `test`
  - nodes
  - edges
  - version metadata
- Despite that, navigating to `?templateId=...` still leaves the canvas empty unless a separate detail fetch succeeds.

Reason this is potentially unreasonable:
- The page already has enough structural data available in adjacent flows to render at least a draft or fallback view.
- Depending solely on a failing detail endpoint makes the edit path brittle.

### 6. Pipeline run modal asset search does not find a known asset id

Observed:
- From saved pipelines, clicking `test -> 运行` opens the expected run modal.
- The modal correctly defaults to:
  - template version `v2`
  - no-asset warning
  - default execution target
- But searching for a known existing asset id `CYB10A01` returns:
  - `未找到匹配的资产`

Why this is suspicious:
- The same session can open `/assets/CYB10A01` successfully.
- So either:
  - the run modal search uses a different API/query mode,
  - the search path rejects direct asset ids incorrectly,
  - or the modal is querying with the wrong contract.

Why this is unreasonable:
- A pipeline run modal that cannot find a canonical, existing asset id undermines asset-based pipeline execution.

API evidence:
- Modal search request:
  - `GET /api/v1/search/assets?q=CYB10A01&page_size=50`
  - `200`
  - response: `items: []`, `total: 0`
- At the same time:
  - `/assets/CYB10A01` opens successfully in the same session

Interpretation:
- The run modal is backed by a search index path rather than a canonical asset lookup path.
- That search path currently misses at least one valid asset id that the asset detail route can resolve.

UI details:
- For `test -> 运行`, the modal opens cleanly and shows:
  - version `v2`
  - default execution target
  - no-asset warning
- But after searching `CYB10A01`, the modal settles on:
  - `未找到匹配的资产`

Additional control-checks:
- The issue is **not** a global asset-search outage.
- In the same session:
  - `SDKT0202` -> asset detail `200`, search total `1`
  - `SDKT0101` -> asset detail `200`, search total `1`
  - `j3QBBiZL` -> asset detail `200`, search total `1`
  - `LQVV2PU2` -> asset detail `200`, search total `1`
- So the mismatch is specifically that `CYB10A01` resolves via asset detail but is absent from search results.

### 7. Multi-version edit defaults to current row version, not active version

Observed:
- Saved pipelines list shows:
  - `pipeline-20260605-174332`
  - row version `v3`
  - `活跃: v1`
- Opening the template edit page by full UUID loads:
  - pipeline name `pipeline-20260605-174332`
  - version selector `v3`

Why this is potentially unreasonable:
- The list UI communicates two competing version concepts:
  - row version
  - active version
- It is not obvious whether edit should open the row version or the active version.

Current status:
- Not marked as a hard bug yet.
- But it is a product/UX ambiguity worth tracking because users may expect “edit” to open the active version.

Additional verified behavior:
- Opening `pipeline-20260605-174332` by full UUID loads `v3` by default.
- The version selector is functional and lists:
  - `v3`
  - `v2`
  - `v1`
- Selecting `v1` changes the route to a different template UUID:
  - `/pipeline?templateId=5baf00d5-3110-480d-bc4f-2af52a082ede`
- After switching, the canvas content changes accordingly and only the `find-toni-stats` node remains visible.

### 8. Historical failed execution detail can briefly degrade before run ledger hydrates

Observed on:
- `/pipeline/executions/pipeline-20260605-190155-5583db?runId=c61a2127-2c62-401d-9925-ffb98cf49d1f`

Early snapshot sometimes showed:
- pipeline name as `—`
- event count `0`
- technical node names `step-step-1`, `step-step-2`

After data hydration, the same page settled to the correct state:
- pipeline name `pipeline-20260605-190155`
- version `v1`
- event count `15`
- business node name `Pass Through`

Why this is worth tracking:
- The final state is correct, so this is not a hard blocking bug.
- But the page visibly passes through a misleading intermediate state, which can look broken in slow dev sessions.

### 9. Failed-node log modal can load but still show no visible log content

Observed on failed node `Pass Through`:
- `查看日志` opens the modal successfully
- metadata loads:
  - node id
  - type `Pod`
  - status `Failed`
  - message `main: Error (exit code 2)`
- but visible log content remains empty:
  - `暂无日志输出`
  - `显示 0 字符 / 0 行`
  - realtime status `未连接`

Why this is suspicious:
- Not every failed step must have useful logs.
- But a failed pod with an exit-code error and zero visible logs is operationally weak for debugging.

### 10. Pipeline save API rejects object-form component env even though component create accepted it

Observed while creating a new complex regression template from browser-side API calls:
- Creating custom component `Chain Stamp` succeeded with:
  - `POST /api/v1/pipeline-components`
  - payload containing object-form env, e.g. `env: { STAGE: 'unset' }`
- Saving a pipeline that embeds component `env` as an object failed:
  - `POST /api/v1/pipelines`
  - `400 INVALID_ARGUMENT`
  - error:
    - `json: cannot unmarshal object into Go struct field Component.nodes.component.env of type []transpiler.EnvVar`

Why this is unreasonable:
- The component registry accepts one env shape.
- The pipeline save/transpiler path expects a different env shape.
- This is a contract mismatch inside the same product surface.

### 11. First complex DAG run failed because command assumed `/tmp/outputs/output` existed

Observed on run:
- `mcp-complex-fanout-chain-9efa2f`

Failure details:
- root node failed
- log excerpt:
  - `tee: /tmp/outputs/output: No such file or directory`

Impact:
- downstream branch nodes were all `Omitted`

Interpretation:
- This dev runtime does not guarantee that manually writing to `/tmp/outputs/output` is safe for every shell-style component.

### 12. Complex DAG regression run can be created and run successfully on dev

Verified working path:
- Created custom component:
  - `Chain Stamp`
  - `POST /api/v1/pipeline-components` -> `201`
- Created complex DAG template:
  - name: `mcp-complex-fanout-chain-ok`
  - node count: `5`
  - edges: `4`
  - structure:
    - `seed-root`
    - `left-a -> left-b`
    - `right-a -> right-b`
  - `POST /api/v1/pipelines` -> `201`
- Dry-run preview:
  - `POST /api/v1/deploy?dryRun=true` -> `200`
  - Argo manifest rendered with expected DAG topology
- Real run:
  - run name: `mcp-complex-fanout-chain-ok-cb17d8`
  - run id: `caa8a727-dedb-42b3-8ab7-7e625dff9c3e`
  - template snapshot: `v2 / 2d5f654a`
- Final observed result:
  - workflow status `Succeeded`
  - event count `34`
  - all five nodes successful:
    - `seed-root`
    - `left-a`
    - `left-b`
    - `right-a`
    - `right-b`

Why this matters:
- This is a real dev E2E pass for a non-trivial multi-branch pipeline.
- It proves the current pipeline path is capable of saving, previewing, submitting, and completing a complex DAG in dev.

### 13. Successful workflow detail can still fall back to technical-node / zero-event state

Observed on successful complex run:
- run name: `mcp-complex-fanout-chain-ok-cb17d8`
- run id: `caa8a727-dedb-42b3-8ab7-7e625dff9c3e`

Good state observed earlier:
- business pipeline name present
- event count non-zero
- node table populated from run ledger

But another visit to the same route later showed a degraded state:
- pipeline name `—`
- event count `0`
- technical node names such as:
  - `step-right-a`
  - `step-right-b`
  - `step-seed`
  - `step-left-a`
  - `step-left-b`
- even though:
  - `GET /api/v1/pipeline-runs/<runId>` returned `200`
  - `GET /api/v1/workflows/<workflowName>` returned `200`

Why this is unreasonable:
- A completed successful run should not regress to a less informative presentation on revisit.
- This suggests a race / hydration bug between workflow data and pipeline-run ledger data.

### 10. Pipeline save API rejects object-form component env even though component create accepted it

Observed while creating a new complex regression template from browser-side API calls:
- Creating custom component `Chain Stamp` succeeded with:
  - `POST /api/v1/pipeline-components`
  - payload containing object-form env, e.g. `env: { STAGE: 'unset' }`
- Saving a pipeline that embeds component `env` as an object failed:
  - `POST /api/v1/pipelines`
  - `400 INVALID_ARGUMENT`
  - error:
    - `json: cannot unmarshal object into Go struct field Component.nodes.component.env of type []transpiler.EnvVar`

Why this is unreasonable:
- The component registry accepts one env shape.
- The pipeline save/transpiler path expects a different env shape.
- This is a contract mismatch inside the same product surface.

## API evidence snapshots

- `GET /api/v1/pipelines` includes:
  - `test.id = c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - `test.version = 2`
  - `test.versionCount = 2`
- `GET /api/v1/pipelines/c00f1136` -> `404`
- `GET /api/v1/pipelines/c00f1136/versions` -> `200`
- `GET /api/v1/pipelines/c00f1136-1f86-4a32-a25d-91bf81557fc8` behavior in UI:
  - no visible canvas rehydration
- `GET /api/v1/workflows/test-77705a/nodes/test-77705a-4151996238/pod` -> `403`
- `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `404`

## Notes

- This file is append-only for the current regression session.
- New pipeline issues found in later Chrome MCP passes should be appended here rather than replacing earlier findings.

## Retest — after PR #171 merge / frontend dev version c0395d42-ce7f-412e-b37f-30e1de45cfdb

Environment:
- Frontend dev: `https://cyber-databrew-dev.cyberorigin.ai`
- Method: Chrome DevTools MCP only
- Session: logged in as dev user

### Confirmed fixed / working

- Success execution detail remains healthy:
  - URL: `/pipeline/executions/test-77705a?runId=4ec0af22-d1eb-4880-9a62-088a8a5951b0`
  - renders `DataBrew 运行`, pipeline `test`, version `v2`
  - node count `2`
  - event count `19`
  - DAG nodes and node detail rows both show business label `Count Lines`
  - key APIs all returned `200`:
    - `GET /api/v1/pipeline-runs/4ec0af22-d1eb-4880-9a62-088a8a5951b0`
    - `GET /api/v1/workflows/test-77705a`
    - `GET /api/v1/pipeline-runs/4ec0af22-d1eb-4880-9a62-088a8a5951b0/events?limit=100`
    - `GET /api/v1/pipeline-runs/4ec0af22-d1eb-4880-9a62-088a8a5951b0/asset-nodes?limit=500`
    - `GET /api/v1/pipeline-runs/4ec0af22-d1eb-4880-9a62-088a8a5951b0/cost-summary`
- Saved pipeline `test -> 编辑` now uses the full UUID and rehydrates correctly:
  - route: `/pipeline?templateId=c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - canvas name is `test`
  - version selector shows `v2`
  - two `Count Lines` nodes are visible
  - APIs:
    - `GET /api/v1/pipelines/c00f1136-1f86-4a32-a25d-91bf81557fc8` -> `200`
    - `GET /api/v1/pipelines/c00f1136-1f86-4a32-a25d-91bf81557fc8/versions` -> `200`
- Failed execution detail no longer visibly degrades to technical `step-step-*` names on first load:
  - URL: `/pipeline/executions/pipeline-20260605-190155-5583db?runId=c61a2127-2c62-401d-9925-ffb98cf49d1f`
  - renders pipeline `pipeline-20260605-190155`, version `v1`
  - event count `15`
  - failed and omitted nodes show business label `Pass Through`
  - key APIs returned `200`
- Successful complex DAG revisit no longer reproduced the earlier zero-event / technical-node fallback:
  - URL: `/pipeline/executions/mcp-complex-fanout-chain-ok-cb17d8?runId=caa8a727-dedb-42b3-8ab7-7e625dff9c3e`
  - renders pipeline `mcp-complex-fanout-chain-ok`, version `v2`
  - node count `5`
  - event count `34`
  - DAG and node detail rows show business labels:
    - `seed-root`
    - `left-a`
    - `left-b`
    - `right-a`
    - `right-b`
  - key APIs returned `200`
- Asset detail `file 文件` tab for `CYB10A01` settles successfully after a brief loading state:
  - shows file registry row `raw_mcap -> CYB10M01`

### Still confirmed issues

- Short-ID pipeline deep links still fail:
  - direct route `/pipeline?templateId=c00f1136` leaves the canvas in new-pipeline empty state
  - `GET /api/v1/pipelines/c00f1136` -> `404`
  - `GET /api/v1/pipelines/c00f1136/versions` -> `200`
  - current interpretation: saved-list edit uses full UUID correctly now; remaining issue is short-ID deep links and inconsistent API contract within the pipeline resource family.
- Pod diagnostics still returns forbidden:
  - UI can open the runtime panel and preserve Argo pod metadata
  - `GET /api/v1/workflows/test-77705a/nodes/test-77705a-4168773857/pod` -> `403`
  - settled UI message: `Pod 诊断数据不可用`
  - console records a 403 resource error
- Pipeline run modal asset search still misses known asset `CYB10A01`:
  - modal shows `未找到匹配的资产`
  - `GET /api/v1/search/assets?q=CYB10A01&page_size=50` -> `200`, `items: []`, `total: 0`
  - `GET /api/v1/assets/CYB10A01` -> `200`
  - control checks in the same session:
    - `SDKT0202` -> search total `1`, detail `200`
    - `SDKT0101` -> search total `1`, detail `200`
    - `j3QBBiZL` -> search total `1`, detail `200`
    - `LQVV2PU2` -> search total `1`, detail `200`
- Asset detail `Action 时间轴` tab still fails:
  - URL: `/assets/CYB10A01`
  - tab settles to `404 请求失败`
  - `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `404`
  - console records a 404 resource error
- Failed-node log modal still loads metadata but no visible log content:
  - failed node: `Pass Through`
  - `GET /api/v1/workflows/pipeline-20260605-190155-5583db/logs?nodeId=pipeline-20260605-190155-5583db-3848181308` -> `200`
  - modal shows `暂无日志输出`
  - visible log count remains `0 字符 / 0 行`
  - node message remains `main: Error (exit code 2)`

### Notes from this retest

- The earlier statement that full UUID edit did not rehydrate is stale after this retest.
- Current saved-pipeline edit behavior should be described as:
  - full UUID edit works
  - short display ID deep-link/API behavior remains broken/inconsistent
- Console issues unrelated to tested behavior were also observed:
  - Ant Design focus warning during tab/modal transition
  - form field missing `id`/`name` accessibility issue

## Current issue queue — from 2026-06-07 retest

These are the problems still confirmed after the PR #171 retest. Use this section as the clean source when filing follow-up work.

### A. Pipeline short-ID deep links / API contract inconsistency

Status:
- Still broken.

Evidence:
- Direct route `/pipeline?templateId=c00f1136` leaves the canvas in new-pipeline empty state.
- `GET /api/v1/pipelines/c00f1136` -> `404`
- `GET /api/v1/pipelines/c00f1136/versions` -> `200`
- Full UUID route works:
  - `/pipeline?templateId=c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - `GET /api/v1/pipelines/c00f1136-1f86-4a32-a25d-91bf81557fc8` -> `200`
  - canvas rehydrates correctly.

Impact:
- Displayed short IDs look usable but are not valid template detail identifiers.
- The pipeline API family is inconsistent: `versions` accepts short ID while detail does not.

### B. Pod diagnostics still forbidden

Status:
- Still broken at API/RBAC level.

Evidence:
- `GET /api/v1/workflows/test-77705a/nodes/test-77705a-4168773857/pod` -> `403`
- UI settles to `Pod 诊断数据不可用`.
- Console records a 403 resource error.

Impact:
- Runtime panel can preserve Argo node metadata, but actual pod diagnostics remain unavailable in dev.

### C. Pipeline run modal cannot find known asset `CYB10A01`

Status:
- Still broken for this canonical asset ID.

Evidence:
- Run modal search for `CYB10A01` shows `未找到匹配的资产`.
- `GET /api/v1/search/assets?q=CYB10A01&page_size=50` -> `200`, `items: []`, `total: 0`
- `GET /api/v1/assets/CYB10A01` -> `200`
- Control checks in the same session:
  - `SDKT0202` -> search total `1`, detail `200`
  - `SDKT0101` -> search total `1`, detail `200`
  - `j3QBBiZL` -> search total `1`, detail `200`
  - `LQVV2PU2` -> search total `1`, detail `200`

Impact:
- Asset-based pipeline execution cannot reliably select an existing asset by direct asset ID.
- This appears to be a search/index or modal lookup contract issue, not a global asset-search outage.

### D. Asset detail `Action 时间轴` tab returns 404

Status:
- Still broken.

Evidence:
- Asset detail route: `/assets/CYB10A01`
- `Action 时间轴` tab settles to `404 请求失败`.
- `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `404`
- Console records a 404 resource error.

Impact:
- User-visible tab is backed by a missing or unrouted endpoint in dev.

### E. Failed-node log modal has metadata but no visible logs

Status:
- Still suspicious / operationally weak.

Evidence:
- Failed execution:
  - `/pipeline/executions/pipeline-20260605-190155-5583db?runId=c61a2127-2c62-401d-9925-ffb98cf49d1f`
- Failed node: `Pass Through`
- `GET /api/v1/workflows/pipeline-20260605-190155-5583db/logs?nodeId=pipeline-20260605-190155-5583db-3848181308` -> `200`
- Modal metadata loads:
  - type `Pod`
  - status `Failed`
  - message `main: Error (exit code 2)`
- Modal still shows:
  - `暂无日志输出`
  - `0 字符 / 0 行`

Impact:
- The failed run is visible, but the log surface gives no useful debugging output beyond node metadata.

## Complex DAG QA run — new custom component and runnable fanout/fanin flow

Environment:
- Frontend dev: `https://cyber-databrew-dev.cyberorigin.ai`
- Method: Chrome DevTools MCP only
- Created through the logged-in browser session, not local curl

### Created component

- Component name: `qa-complex-stage-20260607041627`
- Component ID: `2f5fb0a4-bb99-49cc-9a1f-c1ead55640c5`
- Image: `alpine:3.19`
- Tag: `qa-dev`
- Owner: `ruipeng.huang@cyberorigin.ai`
- API:
  - `POST /api/v1/pipeline-components` -> `201`
- Component behavior:
  - shell container
  - creates `/tmp/outputs`
  - writes declared `output` port safely

### Created complex DAG template

- Template name: `qa-complex-dag-20260607041627`
- Template ID: `5d8ce34a-a285-4cc6-985c-72178c828692`
- Version: `v1`
- Node count: `8`
- Edge count: `9`
- Structure:
  - `seed-root`
  - fanout from `seed-root` to:
    - `left-a -> left-b`
    - `right-a -> right-b`
    - `mid-a`
  - three-way join:
    - `left-b -> join-abc.left`
    - `right-b -> join-abc.right`
    - `mid-a -> join-abc.input`
  - final sink:
    - `join-abc -> final-sink`
- API:
  - `POST /api/v1/deploy?dryRun=true` -> `200`
  - rendered Argo DAG dependencies correctly, including:
    - `step-left-a` depends on `step-seed-root`
    - `step-left-b` depends on `step-left-a`
    - `step-right-a` depends on `step-seed-root`
    - `step-right-b` depends on `step-right-a`
    - `step-mid-a` depends on `step-seed-root`
    - `step-join-abc` depends on `step-left-b`, `step-right-b`, and `step-mid-a`
    - `step-final-sink` depends on `step-join-abc`
  - `POST /api/v1/pipelines` -> `201`

### Real run result

- Run ID: `dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`
- Workflow name: `qa-complex-dag-20260607041627-e3d3c7`
- Trigger: manual no-asset run
- Execution target: `default`
- Namespace: `cyber-databrew-dev`
- API:
  - `POST /api/v1/pipeline-runs/template/5d8ce34a-a285-4cc6-985c-72178c828692` -> `201`
  - poll `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474` -> final `Succeeded`
  - final event count from `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474/events?limit=100` -> `52`
- Poll progression:
  - `Running`, event total `20`
  - `Running`, event total `23`
  - `Running`, event total `35`
  - `Running`, event total `42`
  - `Running`, event total `47`
  - `Succeeded`, event total `52`
- Final duration in UI: about `1m 1s`

### UI verification

- URL:
  - `/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`
- Page renders:
  - status `Succeeded`
  - `DataBrew 运行`
  - pipeline `qa-complex-dag-20260607041627`
  - version `v1`
  - node count `8`
  - event count `52`
  - snapshot `5d8ce34a`
- DAG and node detail rows show business labels:
  - `qa-seed-root`
  - `qa-left-a`
  - `qa-left-b`
  - `qa-right-a`
  - `qa-right-b`
  - `qa-mid-a`
  - `qa-join-abc`
  - `qa-final-sink`
- No technical `step-*` labels were visible in the DAG cards or node detail rows.
- Screenshot:
  - Captured during MCP verification but intentionally not kept in git.
- Console:
  - no runtime error/warn for the run detail page
  - only existing accessibility issue: form field missing `id`/`name`
- Hydration APIs:
  - `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474` -> `200`
  - `GET /api/v1/workflows/qa-complex-dag-20260607041627-e3d3c7` -> `200`
  - `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474/events?limit=100` -> `200`
  - `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474/asset-nodes?limit=500` -> `200`
  - `GET /api/v1/pipeline-runs/dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474/cost-summary` -> `200`

### Runtime evidence from the new run

- Logs work for the new successful complex run:
  - `GET /api/v1/workflows/qa-complex-dag-20260607041627-e3d3c7/logs?nodeId=qa-complex-dag-20260607041627-e3d3c7-377069038` -> `200`
  - log body includes:
    - `qa-final-sink:start`
    - `qa-final-sink:<timestamp>`
- Pod diagnostics is still forbidden for the new run:
  - `GET /api/v1/workflows/qa-complex-dag-20260607041627-e3d3c7/nodes/qa-complex-dag-20260607041627-e3d3c7-377069038/pod` -> `403`
  - body includes:
    - `K8S_FORBIDDEN`
    - `Kubernetes credentials cannot read pod diagnostics`

### Takeaway

- Complex fanout/fanin DAG execution is currently capable of saving, dry-running, submitting, completing, hydrating, and rendering correctly on dev.
- The workflow display-name fix holds on a newly created complex DAG where node IDs and component business names intentionally differ.
- Pod diagnostics remains an environment/RBAC problem independent of whether the workflow itself succeeds.

## CYB-1783 Fix In Progress — 2026-06-07

### Scope

- Linear: `CYB-1783`
- Branch: `fix/CYB-1783-dev-regression-fixes`
- OpenSpec: `openspec/changes/CYB-1783-dev-regression-fixes`

### Code-controlled bugs addressed

- Short pipeline template IDs:
  - Symptom: `/pipeline?templateId=c00f1136` opened an empty new-pipeline canvas while `/api/v1/pipelines/c00f1136` returned `404`.
  - Fix: backend template detail lookup now resolves short display IDs only when they uniquely match one saved template UUID prefix.
- Asset Action timeline route:
  - Symptom: `/api/v1/assets/CYB10A01/action-annotations?limit=200` returned `404`.
  - Fix: backend now serves `/assets/:id/action-annotations[/:action_id]` through the same handler as `/assets/:id/actions[/:action_id]`.
- Pipeline run modal asset search:
  - Symptom: search for `CYB10A01` returned no rows even though `GET /api/v1/assets/CYB10A01` returned `200`.
  - Fix: `AssetPicker` now falls back to direct asset lookup when search returns zero rows for an exact asset ID query.

### Local verification before dev deploy

- `cd backend && go test ./internal/handlers/pipeline/...` -> pass
- `cd backend && go test ./routes` -> pass
- `cd backend && go test ./...` -> pass
- `cd Frontend && npm run test -- --run src/components/pipeline/AssetPicker.test.tsx` -> pass, `12` tests
- `cd Frontend && npm run lint` -> pass
- `cd Frontend && npm run build` -> pass
- `bash -n scripts/verify-api-guide.sh` -> pass

### Pending dev verification

- Deploy backend dev and verify:
  - `GET /api/v1/pipelines/c00f1136` -> `200`
  - `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `200`
- Deploy frontend dev Worker and verify with Chrome DevTools MCP:
  - `/pipeline?templateId=c00f1136` rehydrates saved template `test`.
  - run modal search for `CYB10A01` displays a selectable asset.
  - `/assets/CYB10A01` Action timeline no longer shows the 404 error state.

### Deploy blocker

- Backend docker build completed for image tag:
  - `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:6d4cac9-cyb1783`
- Backend push/deploy failed before reaching Cloud Run:
  - `docker-credential-gcloud` not found in `PATH`
  - `gcloud config config-helper` failed because current auth tokens require interactive reauthentication
  - active account reported by `gcloud auth list`: `ruipeng.huang@cyberorigin.ai`
- Frontend Worker deploy is also blocked:
  - `wrangler` is not installed globally
  - `npx wrangler whoami` works but reports not authenticated and requires `wrangler login`
- Required next local actions before deploy verification can continue:
  - `gcloud auth login`
  - `gcloud auth configure-docker us-central1-docker.pkg.dev`
  - `npx wrangler login`

### Dev deploy and verification complete

- Backend:
  - Image tag: `cyber-databrew-backend:6d4cac9-cyb1783b`
  - Image digest: `sha256:8741db84869ddee09b6e2be5cb79c5282c04177a132798119b6e620297fccb49`
  - Cloud Run revision: `cyber-databrew-backend-dev-cyb1783b2`
  - Dev URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
  - Note: deploy required forcing traffic to `cyb1783b2`; earlier `00630-hmj` remained serving old code until traffic was explicitly updated.
- Frontend:
  - Worker: `cyber-databrew-dev`
  - Version ID: `e5206c26-ce82-4cb4-8914-800c34fad18c`
  - Dev URL: `https://cyber-databrew-dev.cyberorigin.ai/`
- Backend API smoke:
  - `GET /api/v1/pipelines/c00f1136` -> `200`, template `test`, full id `c00f1136-1f86-4a32-a25d-91bf81557fc8`
  - `GET /api/v1/assets/CYB10A01/action-annotations?limit=200` -> `200`, response `{"asset_id":"CYB10A01","items":[],"total":0}`
  - `GET /api/v1/assets/CYB10A01/actions?limit=200` -> `200`, backward-compatible response
- Chrome DevTools MCP:
  - `/pipeline?templateId=c00f1136` rehydrates pipeline `test` with two `Count Lines` nodes.
  - Deploy modal search for `CYB10A01` first calls `/search/assets?q=CYB10A01&page_size=50` -> `200`, then fallback calls `/assets/CYB10A01` -> `200`, and displays a selectable `CYB10A01` row (`segment`, `ready`, `gs://cyb1100/CYB10A01`).
  - `/assets/CYB10A01` Action timeline calls `/assets/CYB10A01/action-annotations?limit=200` -> `200` and shows empty state `该 seg 暂无 action`, not the prior 404 error.
  - Console has no runtime `error` / `warn`; remaining messages are existing browser issues for deprecated feature usage and form fields missing `id` / `name`.
- Screenshots:
  - Captured during MCP verification, but PNG artifacts were removed intentionally and will not be committed.

## CYB-1793 Workflow DAG Layout — 2026-06-07

### Scope

- Linear: `CYB-1793`
- Branch: `fix/CYB-1793-workflow-dag-layout`
- OpenSpec: `openspec/changes/CYB-1793-workflow-dag-layout`

### UI issues addressed

- Complex fan-in DAG layout:
  - Symptom: join nodes rendered off-center relative to their incoming branches, which made the dependency graph feel visually awkward even though the workflow was system-rendered correctly.
  - Fix: after dagre computes the layout, multi-input join nodes are centered over the average vertical center of their direct visible predecessors.
- Fan-in edge readability:
  - Symptom: several incoming branches to the same join read like incidental crossings.
  - Fix: join incoming edges get a slightly stronger visual style while preserving the same DAG semantics.
- Read-only execution handles:
  - Symptom: React Flow handles were hidden visually but could still expose editing affordances.
  - Fix: execution DAG handles are transparent and have `pointer-events: none`.

### Local verification before dev deploy

- `cd Frontend && npm run test -- --run src/pages/WorkflowDagView.test.tsx` -> pass, `4` tests
- `cd Frontend && npm run lint` -> pass
- `cd Frontend && npm run build` -> pass; existing large chunk warnings only

### Dev deploy and Chrome MCP verification

- Frontend:
  - Worker: `cyber-databrew-dev`
  - Version ID: `46345da4-2b70-4747-9099-7f5ba31396a5`
  - Dev URL: `https://cyber-databrew-dev.cyberorigin.ai/`
- Chrome DevTools MCP URL:
  - `https://cyber-databrew-dev.cyberorigin.ai/pipeline/executions/qa-complex-dag-20260607041627-e3d3c7?runId=dcb6cbb1-8e5d-4131-aab0-b99c0f1b4474`
- Chrome DevTools MCP DOM checks:
  - `nodeCount = 8`
  - predecessor centerY values for `qa-left-b`, `qa-right-b`, `qa-mid-a`: `348.2`, `499.1`, `650.0`
  - average predecessor centerY: `499.1`
  - `qa-join-abc` centerY: `499.1`
  - `joinDeltaY = 0`
  - fan-in styled edge count: `3`
  - hidden handle count: `16`
  - visible or clickable handle count: `0`
- Chrome DevTools MCP page checks:
  - Status `Succeeded`, `DataBrew 运行`, `8` nodes, `52` events.
  - Key workflow APIs returned `200`; Cloudflare RUM returned `204`.
  - Console has no runtime `error` / `warn`; remaining messages are existing browser issues for deprecated feature usage and form fields missing `id` / `name`.
- Screenshots:
  - No PNG artifacts were written to the repo.
