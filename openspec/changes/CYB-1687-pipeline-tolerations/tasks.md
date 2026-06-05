## Implementation

- [x] Audit current component resource JSON and pipeline node data flow for where tolerations must be added.
- [x] Add a DataBrew-owned toleration model to transpiler pipeline structs.
- [x] Extend component normalization/read helpers to accept tolerations from JSON resources.
- [x] Add `gpu-l4` preset injection for GPU toleration only.
- [x] Emit node tolerations from `buildContainerTemplate`.
- [x] Emit node tolerations from `buildScriptTemplate`.
- [x] Add component API/types support for tolerations.
- [x] Add component manager UI for editing default tolerations.
- [x] Add pipeline node type/config support for tolerations.
- [x] Ensure newly dragged nodes inherit component tolerations.

## Verification

- [x] Add backend transpiler tests for toleration emission.
- [x] Add backend component normalization tests for toleration JSON and gpu preset behavior.
- [x] Add frontend tests for component toleration editing and node inheritance.
- [x] Run targeted backend tests.
- [x] Run targeted frontend tests.
- [x] Run `cd Frontend && npm run build`.
- [x] Run `git diff --check`.
- [ ] Verify a dev pipeline/component flow against a tainted target after deploy.
