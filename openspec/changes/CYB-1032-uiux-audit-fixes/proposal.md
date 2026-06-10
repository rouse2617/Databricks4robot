# CYB-1032 — UI/UX Audit Fixes: Contrast, Touch Targets, Design Tokens

## Problem

Comprehensive UI/UX audit of the version control subsystem identified systemic accessibility and design system issues:

1. **14 WCAG AA contrast failures** — text colors as low as 1.2:1 ratio (required 4.5:1)
2. **4 touch target violations** — buttons at ~20px height (required 44px)
3. **252+ hardcoded hex colors** project-wide — no design token bridge between Ant Design and Tailwind
4. **Heading hierarchy skip** — `<h4>` with no preceding h1-h3
5. **Dual CSS `:root` conflict** — `index.css` and `design-tokens.css` define competing color palettes

## Proposed Solution

### P0 — Critical (accessibility blockers)
- Fix contrast failures in LogicalAssetId, #999 text across 7+ files, AlgoStatusCell status badges
- Fix heading hierarchy in AssetDetailPage (h4 → h1)
- Resolve dual `:root` CSS conflict by merging design-tokens.css into index.css

### P1 — High priority (functional UX)
- Fix touch target failures on link buttons (remove `p-0 h-auto` overrides)
- Fix VersionHistoryBanner "跳转到当前版" button size
- Fix tab counts showing (0) during loading

### P2 — Design system debt
- Extend Tailwind config with project color tokens
- Replace hardcoded hex values in version control components with design tokens
- Replace VersionHistoryBanner with Ant Design `<Alert type="warning">`

## Scope

**In scope:** Frontend-only changes to 7 files in `Frontend/src/`
**Out of scope:** Backend, SDK, ECharts colors (DashboardPage), AssetsCardView (55+ hex values — separate PR)

## Non-goals
- Full design system overhaul — only fix the version control subsystem
- Dark mode support — not in scope for this change
