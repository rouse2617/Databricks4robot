# Tasks — CYB-1516

## Context files
- `docs-site/docusaurus.config.ts` — navbar labels, site metadata, preset/static handling
- `docs-site/docs/overview.md` — overview page content and next-step entry points
- `docs-site/src/css/custom.css` — link contrast, content width, overview entry styles
- `docs-site/static/` — static metadata files served from the docs base URL
- `Frontend/public/` — root static metadata files served by the main app

## Implementation
- [x] [docs-site] Rename the top-level `Guides` navbar item to a label that matches the documentation entry it opens.
- [x] [docs-site] Add visible overview next-step links near the top of `docs-site/docs/overview.md`.
- [x] [docs-site] Adjust CSS variables/link states so breadcrumb and article links pass contrast checks.
- [x] [docs-site] Tune desktop docs layout width without introducing mobile overflow.
- [x] [docs-site] Add valid `robots.txt` and `llms.txt` static files.
- [x] [Frontend] Add root-level valid `robots.txt` and `llms.txt` static files.

## Verification
- [x] Run `cd docs-site && npm run build`.
- [x] Run local docs preview/server and inspect `/doc/overview` with Chrome DevTools MCP on desktop and mobile.
- [x] Run frontend build or targeted static asset verification for root metadata files.
- [x] Re-run Lighthouse snapshot checks for accessibility and SEO.
- [x] Verify `/doc/robots.txt` and `/doc/llms.txt` are served as plain static files in the docs build.
- [x] Verify root `/robots.txt` and `/llms.txt` are served as plain static files in the frontend build.

## Deploy verification
- [ ] After frontend/docs deploy, open `https://cyber-databrew.cyberorigin.ai/doc/overview` and verify the corrected navigation labels and overview entry points.
- [ ] Verify `https://cyber-databrew.cyberorigin.ai/doc/robots.txt` and `https://cyber-databrew.cyberorigin.ai/doc/llms.txt` return valid static content.
- [ ] Verify `https://cyber-databrew.cyberorigin.ai/robots.txt` and `https://cyber-databrew.cyberorigin.ai/llms.txt` return valid static content.
