# Tasks — CYB-3094 / CYB-3095

## CYB-3094 — component form ≥1 port invariant
- [x] Disable the port remove button when only one port row remains (input and
      output), with an explanatory tooltip (`ComponentManager.tsx` `PortFormList`).
- [x] Regression test: create form's last input/output remove buttons are disabled;
      adding a second port re-enables removal (`ComponentManager.test.tsx`).

## CYB-3095 — exclude onExit hook from DAG view + count
- [x] Add exit-notify exclusion to `isDisplayableNode` (`WorkflowDagView.tsx`) so DAG
      rendering, `countDisplayableWorkflowNodes`, the top-level node count, and the
      timeline all skip the hook.
- [x] Regression test: onExit hook excluded by templateName and by `.onExit` name
      suffix (`WorkflowDagView.test.tsx`).

## Verification
- [x] Tier L: targeted `npm run test -- --run` (11 passed), `biome check` clean, `npm run build` clean.
- [x] Chrome DevTools MCP e2e on local frontend (working tree) against Cloud Run dev backend:
      CYB-3094 create form last-port remove disabled + re-enabled after adding a port;
      CYB-3095 run `58c1d118` DAG/step count/top node count all = 1, no `.onExit` node, no console errors.
- [ ] PR to dev referencing CYB-3094 + CYB-3095.
- [ ] Post-deploy confirmation on deployed dev revision.
