## 2026-05-24 — OpenSpec approved
- **Context**: Code quality cleanup after CYB-1162 review
- **Decision**: Implement 5 small fixes across 4 files:
  1. ValidateCreate reject unknown asset_type (asset_validator.go)
  2. Remove unreachable code in Emitter (emitter.go)
  3. Prune ParentInfo to only used fields (asset_validator.go)
  4. Log customerID query errors (delivery_eligibility_projector.go)
  5. Add derived_asset parent-not-found test (asset_validator_test.go)
