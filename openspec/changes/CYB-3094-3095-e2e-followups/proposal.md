# CYB-3094 / CYB-3095 — e2e walkthrough follow-up UI fixes

## Problem

Two UI/UX inconsistencies surfaced during the create-component → DAG → run e2e walkthrough:

1. **CYB-3094** — In the 新建组件 form, removing all input/output ports appears to
   succeed, but the placed canvas node still shows the default `input`/`output`
   ports. The whole pipeline stack (form save, component read, add-node, deploy
   serialization) assumes every node has ≥1 input and ≥1 output port and silently
   re-injects defaults, so the form's "remove the last port" gesture never sticks —
   the form was misleading the user.

2. **CYB-3095** — A successful run's detail DAG view shows the CYB-3058
   `databrew-exit-notify` onExit hook node as a second "step" (and counts it as a
   node), even though the 节点明细 table correctly excludes it. The DAG view builds
   from the raw Argo node status, which still includes the infrastructure hook.

## Scope

- Frontend only.
- CYB-3094: make the component form honest about the ≥1-port invariant — forbid
  removing the last input port and the last output port (disable the remove control
  with an explanatory tooltip).
- CYB-3095: exclude the onExit notify hook (`templateName === "databrew-exit-notify"`
  or name ending in `.onExit`) from `isDisplayableNode`, the single convergence point
  behind DAG rendering, the "N 个步骤" count, the top-level node count, and the
  timeline view.

## Out of scope

- True zero-port components (would require unwinding the ≥1-port invariant across ~5
  frontend layers plus deploy serialization; deferred — see decisions.md).
- Backend changes (backend already excludes exit-notify from the step list via
  `runNodesFromWorkflow`, and already preserves empty ports on the base component path).
