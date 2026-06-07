# Tasks - CYB-1796

## Context Files

- `Frontend/vite.config.ts` - app version/build/environment compile-time constants.
- `Frontend/package.json` - frontend build scripts.
- `wrangler.jsonc` - Worker static assets binding and dev Worker name.
- `_worker.js` - Worker static assets and API proxy behavior.
- `docs/agents/deploy-before-commit.md` - canonical frontend deploy workflow.
- `docs/agents/deploy-verification.md` - frontend dev verification workflow.

## Implementation

- [x] [Frontend] Add a dev-specific build command that sets `VITE_APP_ENV=dev`.
- [x] [scripts] Add or update a canonical dev Worker deploy script that clears `site/`, builds dev assets, copies fresh `Frontend/dist`, and deploys `wrangler --env dev`.
- [x] [docs] Update deploy guidance to use the canonical dev Worker command.
- [x] [docs] Add explicit verification for served `index-*.js` and sidebar environment marker.

## Verification

- [x] [Frontend] `npm run lint`
  - Result: pass, `Checked 237 files`.
- [x] [Frontend] `npm run build`
  - Result: pass; existing large chunk warnings only.
- [x] [Frontend] Run dev-specific frontend build and verify `Frontend/dist/assets/index-*.js` embeds `environment: "dev"`.
  - Command: `DEPLOY_WORKER=0 INCLUDE_DOCS=0 bash scripts/deploy-frontend-dev-worker.sh`
  - Result: `site/index.html` and `Frontend/dist/index.html` both reference `assets/index-XbBkNF9n.js`; `site/assets/index-XbBkNF9n.js` embeds `environment:w("dev")`.
- [x] Deploy frontend dev Worker.
  - Command: `bash scripts/deploy-frontend-dev-worker.sh`
  - Worker: `cyber-databrew-dev`
  - Version ID: `fe01278a-a27a-42eb-88f9-fca7f291f0a4`
  - URL: `https://cyber-databrew-dev.cyberorigin.ai/`
- [x] Verify `curl https://cyber-databrew-dev.cyberorigin.ai/` references the freshly built `index-*.js`.
  - Result: served HTML references `/assets/index-XbBkNF9n.js`; served bundle embeds `environment:w("dev")`.
- [x] Chrome DevTools MCP verifies sidebar marker shows `v0.1.1 · <sha> · dev`.
  - Result: dashboard marker shows `v0.1.1 · 736bb98-dirty · dev`.
- [x] Chrome DevTools MCP console/network check has no new runtime errors.
  - Result: key dashboard APIs returned `200`; console has no runtime `error` / `warn`, only existing browser issue entries.

## Documentation / Tracking

- [ ] Record deploy evidence in this `tasks.md`.
- [ ] PR body references CYB-1796 and this OpenSpec change.
