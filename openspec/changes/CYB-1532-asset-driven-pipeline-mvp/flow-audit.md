# Flow Audit — CYB-1532

## Environment
- Dev frontend: `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app`
- Date: 2026-06-01 Asia/Shanghai
- Branch baseline: `origin/dev` at `8d4ccd1`

## Flow 1: Component -> Canvas -> Save -> Run
- Created component `cyb1532-user-echo`.
- Dragged it into the design canvas.
- Saved template `cyb1532-user-flow-1`.
- Ran it from the saved pipeline list.
- Result: Argo workflow `cyb1532-user-flow-1-2713e0` was submitted and reached `Error`.

### Finding
The command printed `cyb1532-flow-ok`, but the workflow failed because Argo expected `/tmp/outputs/output` and the component did not write it. The UI allowed this component to be created and run without explaining the output contract.

## Flow 2: Valid Output Component -> Asset Run -> Logs/Resources
- Created component `cyb1532-valid-output-20260601154335`.
- Created template `cyb1532-asset-run-valid-1`.
- Submitted with assets `SDKT0202`, `SDKT0101`, `SDKT0001`.
- Result: Argo workflow `cyb1532-asset-run-valid-1-ccbef8` succeeded.
- Node detail showed pod name, host, status, duration, and resource duration.
- Logs showed `cyb1532-valid-output` and Argo parameter save success.

### Finding
The backend does inject selected asset IDs and asset metadata into workflow env vars. The frontend execution list only links one associated asset, not the selected asset set, and does not show target/namespace.

## Flow 3: Saved Pipeline Primary Run Button
- Opened `流水线` tab and clicked the primary `运行` button on `cyb1532-asset-run-valid-1`.
- Result: a new run started immediately without opening asset selection.

### Finding
The main action creates a no-asset run by default, while asset-based run is hidden behind the small dropdown. This conflicts with the future asset-driven product model.

## UI/UX Findings
- Template CRUD and run history share one long panel; at 100+ records this is hard to scan.
- The primary run action is immediate no-asset run; asset run is secondary and hidden.
- Execution records do not show execution target, cluster, namespace, or service account.
- Asset-bound execution records expose only one `查看关联资产` link, even when the run has multiple assets.
- Node log text is visually concatenated in the modal, making multi-line logs difficult to read.
- Component authoring does not validate image/tag or explain output-file requirements.
- Design canvas component list updates after refresh/loading, but no explicit reload action is visible in the palette.

## Product Implications
- DataBrew needs explicit surfaces for component library, pipeline templates, pipeline runs, and execution targets.
- `pipeline_run_nodes` should be the user-level explanation of "each step ran as a pod", with pod/resource/log fields derived from Argo.
- Asset x node state can be introduced incrementally: first attach asset set to run, then fan out or derive per asset x node state when the pipeline execution model supports per-asset tasks.
