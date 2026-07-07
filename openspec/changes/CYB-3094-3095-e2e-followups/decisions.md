## 2026-07-07 — CYB-3094 fix direction: enforce ≥1 port (not zero-port support)

- **Context**: Removing all component ports doesn't stick; defaults are re-injected
  at ~5 frontend layers (`toPayload`, `toFormValues`, `componentToRegistered`,
  `releaseToRegistered`, `NodeConfigPanel`) plus deploy serialization
  (`canvas-to-dsl`). The backend base create/update path preserves empty ports, but
  the CI release path and the entire frontend assume ≥1 input and ≥1 output port.
- **Decision**: Make the form honest about the existing ≥1-input/≥1-output invariant
  by forbidding removal of the last port. One-file, zero-risk change that fully
  achieves "form == canvas" consistency.
- **Alternatives**: (A) Honor true zero-port components end-to-end — rejected for now:
  broad, deploy-adjacent change for a backlog / no-priority cosmetic bug, and it would
  still leave a save→reload round-trip re-defaulting inconsistency. Reversible — can
  expand to (A) if a real need for portless components appears.
- **Rationale**: Minimal, low-regret, fully consistent; matches the system's
  deeply-baked invariant. Aligns with the project's minimal-infra preference.
- **Note**: Offered the user the A-vs-B direction choice via AskUserQuestion; they were
  away (60s timeout), so proceeded with B per best judgment and flagged the A
  alternative in the delivery summary for review.
