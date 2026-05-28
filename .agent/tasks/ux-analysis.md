# Task: UX/UI Analysis — Pipeline Creation & Deployment Flow

## Background

Review the pipeline creation canvas and deployment flow in the cyber-databrew frontend. This is a single-page app with:
- A **canvas** (React Flow) where users drag pipeline component nodes, connect them with edges
- A **deploy button** that submits the pipeline to Argo Workflows (backend)
- A **deploy success modal** with workflow name and "View Workflow" link
- A **workflow list page** (table) showing past/active workflow runs
- A **workflow detail page** (DAG view + timeline view) with node status and per-node log viewer

## Environment

- **Frontend URL:** https://cyber-databrew-frontend-dev-uc.a.run.app
- **Dev token instruction:** Login button at top-right of the page header, use dev token for auth
- **Branch:** `feat/pipeline-integration`

## Task: Conduct a Professional UX/UI Review

Act as a senior UX/product designer. Review every screen in the pipeline creation → deploy → monitor flow. Be thorough and specific — point out concrete issues with CSS selectors / component names / layout measurements.

### Areas to Cover (be comprehensive):

#### 1. Pipeline Creation Canvas
- Component panel: Is the component list clear? Are component names/descriptions intuitive?
- Search/filter in component panel
- Drag & drop feedback: visual affordance, cursor changes, snap behavior
- Node selection on canvas: visual state, handle visibility
- Edge creation: feedback, constraint clarity (can you connect incompatible nodes?)
- Empty state: what does the canvas look like with 0 nodes? Is there guidance text?
- Node configuration: after placing a node, how do you configure it? Parameters form?
- Save template functionality

#### 2. Deploy Flow
- Deploy button state: disabled vs enabled, what triggers enablement?
- If there's a deploy modal/dialog: check content, clarity, asset selection UX
- Loading states during deploy
- Success/failure feedback: notification, modal, inline error
- Error messages: are they actionable? Do they tell the user what to fix?

#### 3. Workflow List Page
- Table columns: are the right fields shown? Is status color-coded?
- Refresh mechanism: auto-refresh or manual?
- Empty state: "no workflows" message
- Filtering: status filter UX
- Navigation to detail: click row vs button

#### 4. Workflow Detail Page
- DAG visualization: node styling, edge routing, status colors
- Timeline view: time axis, bar chart, node duration
- Node selection: side panel with details tab and logs tab
- Log viewer: loading state, empty log state, error state
- Tab switching: preserved state, performance
- Navigation back to list

#### 5. Cross-Cutting Concerns
- Loading states across all pages
- Error states and recovery paths
- Empty/null/edge case states
- Performance perception: skeleton screens, transitions
- Accessibility: keyboard navigation, ARIA labels, focus management
- Consistency with the rest of the app (colors, spacing, typography, component library usage)
- Mobile/responsive (if applicable)
- Information density: too much/little info at each step?
- Micro-interactions: transitions, animations, status updates

### For Each Issue Found:
1. **Severity:** Critical / Major / Minor / Enhancement
2. **Current behavior:** What happens now (be specific — component name, CSS class, state)
3. **Problem:** Why it's bad (UX principle violated, user confusion point, accessibility violation)
4. **Recommendation:** Concrete fix (what to change, proposed layout, interaction pattern)
5. **Effort estimate:** Small / Medium / Large

### Deliverable

Write findings to `docs/review/ux-analysis-report.md` in the project repo. Structure:

```markdown
# UX/UI Analysis Report — Pipeline Flow

## Summary
(Overall assessment, top 3-5 critical issues, overview of strengths)

## Methodology
(How you reviewed: browser navigation, DevTools inspection, interaction testing)

## Findings

### P1 (Critical)
| # | Screen | Issue | Severity | Fix | Effort |
|---|--------|-------|----------|-----|--------|

### P2 (Major)
...

### P3 (Minor / Enhancement)
...

## Quick Wins (Can fix in < 30 min)
## Deep Dives (Need design iteration)
## Recommendations Summary
```

### Tools Available
- Chrome DevTools: for inspecting DOM, CSS, accessibility tree
- Screenshots: capture visual evidence for key issues
- Console: check for JS errors
- Network panel: check API call latency

### Important Notes
- Do NOT make any code changes — this is analysis only
- Take screenshots where helpful and save to `docs/review/screenshots/`
- Test with the dev backend (frontend should already be pointing to the live backend)
- The auth token is: source the dev-backend-env.sh script or use the login button

Proceed systematically: canvas → deploy → workflow list → workflow detail.
Use `--chrome` flag for browser testing.
