## 2026-06-04 — Frontend dev deploy recovered after registry retry
- **Context**: CYB-1625 touches frontend files, so frontend dev deploy was attempted after backend verification.
- **Decision**: Backend dev was deployed and verified; frontend dev deploy initially failed while fetching the Artifact Registry OAuth token, then succeeded on retry.
- **Alternatives**: Keep retrying the local Docker push, or add a Cloud Build path for the frontend deploy script.
- **Rationale**: The feature-critical runtime fix is backend-side cost and run-ledger behavior. Chrome DevTools verification against the deployed dev frontend and backend confirmed the complex workflow detail page renders the expected 5 business nodes and cost data with no console errors.
