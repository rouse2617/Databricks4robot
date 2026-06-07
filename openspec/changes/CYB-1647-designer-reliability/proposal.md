# Proposal — CYB-1647

## Why
Pipeline authoring still has two high-friction interactions: generated names can
be accidentally appended to, and drag/drop is the only obvious add-node path.
These issues create durable bad template names and weak regression coverage.

## What Changes

### New Capabilities
- Pipeline designer users can add a component through a visible deterministic
  add action on each palette item.
- Pipeline designer tests can verify component insertion through the same
  click-add path users see.

### Modified Capabilities
- Editing an untouched generated pipeline name replaces the generated value
  instead of appending to it.
- The component palette keeps drag/drop available, but the visible primary
  affordance no longer relies on drag/drop precision.

## Impact
- **Affected code**:
  - `Frontend/src/pages/PipelinePage.tsx`
  - `Frontend/src/pages/PipelinePage.test.tsx`
  - `Frontend/src/components/pipeline/ComponentPalette.tsx`
  - `Frontend/src/styles/pipeline.css`
- **New APIs**: none.
- **Dependencies**: none.

## Scope
- **In scope**:
  - Fix generated-name replacement semantics on the designer name input.
  - Add a visible add button or equivalent visible add affordance to palette items.
  - Add focused regression tests for name replacement and click-add behavior.
  - Verify the designer in dev with Chrome DevTools MCP after deploy.
- **Out of scope**:
  - Rewriting React Flow drag/drop internals.
  - Backend pipeline template or run API changes.
  - Component registry CRUD or command template correctness.
  - Execution detail diagnostics beyond the designer entry path.

## Success Criteria
- [ ] Focusing an untouched generated pipeline name and typing a new name saves
  exactly the typed name, without the generated prefix.
- [ ] User-edited or loaded template names are not unexpectedly auto-replaced on
  later focus.
- [ ] Each palette item exposes a visible add action that creates the exact
  selected component.
- [ ] Adding a component through the visible action enables designer save/run
  controls without changing the browser route or requiring drag/drop.
- [ ] Chrome DevTools MCP on dev verifies the click-add and name replacement
  paths with no console errors.

## Goals (SLO)
- **Quality**: Focused frontend tests cover the two changed interaction paths.
