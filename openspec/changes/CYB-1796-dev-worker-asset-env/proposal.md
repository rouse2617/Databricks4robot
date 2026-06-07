# Proposal - CYB-1796

## Why

The dev frontend Worker is serving a bundle that identifies itself as `production`, which makes dev QA evidence ambiguous and suggests the deployed static assets are not being built or published through a reliable dev-specific path.

## What Changes

### New Capabilities

- A dedicated dev Worker build/deploy path ensures the SPA bundle is built with the `dev` environment marker before assets are copied into `site/`.
- Dev verification can assert that the served HTML references the freshly built asset bundle and that the sidebar marker reports `dev`.

### Modified Capabilities

- Frontend Worker deployment for `cyber-databrew-dev` no longer relies on a generic production-mode Vite build for environment identification.
- The static asset copy step no longer leaves stale files in `site/` that can confuse deployment evidence.

## Impact

- **Affected code**: `Frontend/package.json`, `scripts/`, `docs/agents/deploy-before-commit.md`, `docs/agents/deploy-verification.md`
- **New APIs**: none
- **Dependencies**: none

## Scope

- **In scope**: dev Worker frontend build/deploy command, static asset preparation, documentation for the canonical dev deploy path, Chrome DevTools MCP verification.
- **Out of scope**: backend routing, production Worker environment behavior, dashboard/page feature changes, Cloudflare account-level cache policy changes unless required by verification.

## Success Criteria

- [ ] The dev Worker build path embeds `__APP_ENV__ = "dev"` in the deployed frontend bundle.
- [ ] The served dev HTML references the freshly built `index-*.js` asset from the same deploy.
- [ ] The sidebar version marker on `https://cyber-databrew-dev.cyberorigin.ai/` shows `dev`.
- [ ] Chrome DevTools MCP verifies no runtime `error` / `warn` on the deployed dev UI.
