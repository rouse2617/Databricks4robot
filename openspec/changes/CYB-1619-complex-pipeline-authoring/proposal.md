# CYB-1619 — Complex Pipeline Authoring

## Problem

CYB-1614 added important runtime guardrails, but complex pipeline authoring still relies too much on implicit knowledge:

- Component input/output ports are not first-class enough in the UI.
- Users can understand that a pipeline is invalid only after validation, not while composing it.
- Fan-out/fan-in requires knowing that a join node needs distinct target input ports.
- Standard complex pipelines are not available as reusable templates or examples.

This keeps the product at "can run with expert help" instead of "users can build complex workflows themselves".

## Goals

- Let users manage component inputs and outputs from the component UI.
- Make output file contracts visible and hard to misuse.
- Improve fan-in authoring so users can connect multiple upstream nodes into a join node without trial-and-error.
- Add standard complex pipeline examples/templates that demonstrate valid sequential and fan-out/fan-in workflows.

## Non-Goals

- No new database schema unless existing component/template persistence cannot store the required fields.
- No GCP Billing reconciliation in this change.
- No new Argo execution backend; keep using the current transpiler and workflow submission path.
- No mobile-specific UX work.

## Risks

- If the UI creates new port shapes that the backend cannot persist or transpile, saved templates may drift from run behavior.
- If sample templates create long-running pods by default, local/dev testing can become noisy or expensive.
- If fan-in UI is too ambitious, it may delay the simpler P0 need: distinct input ports and clear connection feedback.

## Acceptance

- A user can create/edit component input and output ports without JSON editing.
- The component UI explains `/tmp/outputs/<port>` for outputs, and generated examples follow the contract.
- The pipeline designer makes duplicate target input connections difficult or immediately visible.
- A user can load/create at least three standard complex pipeline examples:
  - Sequential chain.
  - Fan-out/fan-in join.
  - Multi-step observation workflow with 10s+ nodes.
- Existing CYB-1614 validations remain active.
