# Tasks — CYB-1556 Node Detail UI

## Frontend

- [x] Replace abstract "节点" wording with "步骤" in pipeline version/run UI where user-facing.
- [x] Add compact action icons to DAG node cards for logs, runtime, outputs, and details.
- [x] Show visible failure/message summaries on failed/error DAG node cards.
- [x] Reduce workflow node detail drawer top-level tabs to overview, logs, runtime, and input/output.
- [x] Wire node card actions so each icon opens the drawer on the matching section.

## Verification

- [x] Frontend lint/test for touched workflow components.
- [x] Browser/MCP smoke on local workflow detail and run dialog.
