## Linear Issue

<!-- CYB-xxx (required) — https://linear.app/cyberorigin/issue/CYB-xxx -->

## OpenSpec Change

<!-- openspec/changes/CYB-xxx-description/ (required for runtime changes) -->

## Change Type

- [ ] Bug fix
- [ ] Feature
- [ ] Hotfix (spec to follow within T+2)
- [ ] Docs only
- [ ] Infra/CI only

## Scope

- [ ] Backend
- [ ] Frontend
- [ ] SDK
- [ ] Database migration
- [ ] Off-limits zone (requires second reviewer)

## Code Review Checklist

- [ ] Implementation matches Linear Issue requirements; core logic is correct
- [ ] Code style consistent; clear naming; no duplication
- [ ] New dependencies declared; DB changes have migration scripts
- [ ] All CI/CD checks pass
- [ ] Security assessment done (input validation, auth boundaries)
- [ ] Gemini Review High Priority items addressed with reply

## Agent decisions

<!-- Required for hotfix, off-limits, rule conflicts, or skipped MCP deploy verify. Link openspec/.../decisions.md -->

## Test Evidence

<!-- lint/test/build output; command snippets -->

## Deploy Verification

<!-- Required for runtime changes before merge -->

- **Environment:** Cloud Run dev
- **Image tag(s):** backend `cyber-databrew-backend:<sha>` / frontend `cyber-databrew-frontend:<sha>`
- **Revision(s):** backend `cyber-databrew-backend-dev-…` / frontend `cyber-databrew-frontend-dev-…`
- **Dev URL(s):** …

### Frontend（本 PR 改动含 `Frontend/` 时必填）

- [ ] Chrome DevTools MCP: targeted steps on dev (not delegated to user)
- [ ] §6 fixed regression: __/11 pages (see deploy-verification.md)
- [ ] Screenshot(s): `openspec/changes/CYB-xxx-…/deploy-verify-*.png` or attached
- [ ] Console: no new errors

### Backend (if applicable)

- [ ] L1/L2 smoke: command + pass count
- [ ] Targeted API curls for this change
