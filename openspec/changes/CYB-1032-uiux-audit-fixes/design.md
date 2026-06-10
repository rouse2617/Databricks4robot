# Technical Design — CYB-1032

## Color Token Architecture

### Current state
- Ant Design ConfigProvider defines semantic tokens in `main.tsx` (colorPrimary, colorSuccess, etc.)
- Tailwind config has empty `theme.extend` — forces `text-[#xxx]` arbitrary values everywhere
- Two competing CSS `:root` blocks (`index.css` and `design-tokens.css`)

### Target state
1. **Single source of truth**: `index.css` `:root` block defines CSS custom properties
2. **Tailwind bridge**: `tailwind.config.js` extends colors from CSS variables
3. **Ant Design**: ConfigProvider tokens remain unchanged (already correct)
4. **Components**: Use Tailwind semantic classes (`text-primary`, `bg-surface`) or Ant Design `Typography.Text type="secondary"`

### Token mapping

| Semantic name | CSS variable | Tailwind class | Ant Design token | Hex |
|---|---|---|---|---|
| primary | `--color-primary` | `text-primary` | colorPrimary | #2563eb |
| success | `--color-success` | `text-success` | colorSuccess | #16a34a |
| warning | `--color-warning` | `text-warning` | colorWarning | #d97706 |
| error | `--color-error` | `text-error` | colorError | #dc2626 |
| text | `--color-text` | `text-text` | colorText | #1e293b |
| text-secondary | `--color-text-secondary` | `text-text-secondary` | colorTextSecondary | #64748b |
| surface | `--color-bg` | `bg-surface` | colorBgLayout | #f8fafc |
| border | `--color-border` | `border-border` | colorBorder | #e2e8f0 |

### Approach: Tailwind + CSS variables

```js
// tailwind.config.js
theme: {
  extend: {
    colors: {
      primary: 'var(--color-primary)',
      success: 'var(--color-success)',
      warning: 'var(--color-warning)',
      error: 'var(--color-error)',
      text: 'var(--color-text)',
      'text-secondary': 'var(--color-text-secondary)',
      surface: 'var(--color-bg)',
      border: 'var(--color-border)',
    }
  }
}
```

This lets components use `text-primary` instead of `text-[#2563eb]`, while CSS variables provide runtime theming capability.

## Component Changes

### VersionHistoryBanner → Ant Design Alert
Replace entire custom component with `<Alert type="warning" action={...}>`. Eliminates 14 hardcoded colors.

### LogicalAssetId
- Replace `text-[#94A3B8]` (3.05:1) with `text-text-secondary` (4.63:1)
- Remove redundant `title` attribute

### VersionProvenanceTab
- Replace `p-0 h-auto` on link buttons to preserve touch targets
- Add `<ol role="list">` for version chain semantics
- Replace hardcoded hex with Tailwind semantic classes

### AssetDetailPage
- Fix heading: `level={4}` → `level={1}` with Tailwind styling
- Replace `text-[#E2E8F0]` pipe separator with `text-border`

### Tab loading counts
- Show `...` or omit count during loading instead of `(0)`
