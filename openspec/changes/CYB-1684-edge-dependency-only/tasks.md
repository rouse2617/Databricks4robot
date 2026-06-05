# Tasks — CYB-1684

## Implementation

- [x] Make canvas edges generate DAG dependencies only.
- [x] Remove implicit upstream-output-to-downstream-input task arguments for ordinary edges.
- [x] Keep task ordering behavior unchanged.

## Verification

- [x] Add or update transpiler tests for dependency-only edges.
- [x] Run targeted backend tests.
- [x] Run formatting/vet checks for touched backend code.
