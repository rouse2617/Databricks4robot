# Spec delta — delivery-rules

## ADDED: Pre-delivery rule check

**Priority**: P1  
**Rationale**: Prevent non-compliant assets entering a delivery for a customer.

### Scenario: Block PII asset on commit

- **GIVEN** customer `C` has active block rule matching `tag.compliance.pii eq true`
- **AND** asset `A` has human tag `compliance.pii=true`
- **WHEN** operator `POST /deliveries` with `customer_id=C` and `asset_ids=[A]`
- **THEN** response is 422 `DELIVERY_RULE_FAILED`
- **AND** body lists `A` and the blocking `rule_id` / `rule_name`

### Scenario: Legacy delivery without rules

- **GIVEN** no active block rules for customer
- **WHEN** `POST /deliveries` with valid customer and assets
- **THEN** delivery commits as today (201)
