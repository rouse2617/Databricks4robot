# Tasks — CYB-1679

## Implementation

- [x] Audit the current `resourcesDuration` to cost conversion and document which fields are safe to use.
- [x] Remove the unsafe default fallback to `g2-standard-16` / `nvidia-l4` when node metadata is missing.
- [x] Make cost estimation conservative when Argo metadata is incomplete, instead of returning inflated totals.
- [x] Keep execution-list and run-detail cost consumers aligned with the corrected backend value.
- [x] Add regression tests for non-GPU short runs, explicit GPU runs, and missing metadata cases.

## Verification

- [x] Run targeted backend tests for pipeline cost estimation.
- [x] Run relevant frontend tests if any list/detail formatting changes are needed. (No frontend code change required.)
- [ ] Verify a representative run in dev/local no longer shows unrealistic total estimated cost.
- [ ] Open PR to `dev` with Linear and OpenSpec links.
