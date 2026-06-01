# Proposal — CYB-1516

## Why
The public documentation overview page still has an ambiguous top-level navigation label, weak first-step guidance, accessibility contrast failures, and invalid crawler metadata.

## What Changes

### New Capabilities
- The documentation site exposes valid crawler metadata files for `robots.txt` and `llms.txt`.
- The overview page surfaces primary next-step links near the top of the page.

### Modified Capabilities
- Documentation navigation labels match their destination pages.
- Documentation links and breadcrumbs meet readable contrast expectations.
- The desktop documentation layout uses available width more effectively while keeping article text readable.

## Impact
- **Affected code**: `docs-site/docusaurus.config.ts`, `docs-site/docs/overview.md`, `docs-site/src/css/custom.css`, `docs-site/static/*`, `Frontend/public/*`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: public docs UI/UX polish, the `Guides` top-nav label, overview entry points, color contrast, docs layout width, `/doc/robots.txt`, `/doc/llms.txt`, root `/robots.txt`, root `/llms.txt`
- **Out of scope**: backend APIs, SDK behavior, application dashboard UI, content rewrites beyond the overview page

## Success Criteria
- [ ] The top navigation uses a label that clearly describes the documentation entry it opens.
- [ ] Overview gives users visible next-step links before the long capability list.
- [ ] Lighthouse no longer reports the observed link/breadcrumb color contrast failures.
- [ ] `/doc/robots.txt` returns valid robots syntax instead of the HTML shell.
- [ ] `/doc/llms.txt` returns Markdown with an H1 and useful documentation links.
- [ ] Root `/robots.txt` and `/llms.txt` return valid static files for Lighthouse and crawlers.
- [ ] Desktop and mobile documentation views have no obvious overlap or horizontal overflow.
