# Task: Integrate @ant-design/pro-flow

Upgrade the pipeline canvas with @ant-design/pro-flow for better UX.

## Background
Our pipeline canvas currently uses bare @xyflow/react. @ant-design/pro-flow is an Ant Design wrapper that adds:
- 5 selection state styles (SELECT/DANGER/WARNING + sub variants)
- Keyboard shortcuts (Cmd+Z undo, Cmd+Shift+Z redo, Cmd+A select all, Cmd+C/V copy/paste)
- Right-click context menu
- Inspector panel (right-side Drawer)
- Dagre auto-layout
- Zustand state management with Yjs undo/redo

## Reference
- Source at `/Users/rick/src/reference/pro-flow/`
- Our canvas at `Frontend/src/pages/PipelinePage.tsx`

## What to Do

### 1. Install
```bash
cd Frontend && npm install @ant-design/pro-flow
```

### 2. Replace or wrap existing React Flow
The main canvas is in `Frontend/src/pages/PipelinePage.tsx`. We need to integrate pro-flow's `FlowEditor` or `FlowView` component.

Approach: Don't replace everything at once. Instead:
- Wrap the canvas in `FlowEditorProvider` from pro-flow
- Get keyboard shortcuts and selection styles for free
- Keep our existing node types and edge rendering

### 3. Add Keyboard Shortcuts
- Cmd+Z → undo canvas operation
- Cmd+Shift+Z → redo
- Cmd+A → select all nodes
- Cmd+C / Cmd+V → copy/paste nodes

### 4. Add Right-Click Context Menu
- On canvas: paste, select all, zoom options
- On node: delete, configure, copy

### 5. Add Node Selection Styles
- Selected node gets blue border (Ant Design primary color)
- Connected edges get highlighted
- Use pro-flow's built-in `selectType` system

### 6. Verify
```bash
cd Frontend && npm run build
npm run lint
```

## Steps
1. Read existing PipelinePage.tsx to understand current canvas setup
2. Read pro-flow docs and examples
3. Install pro-flow
4. Integrate progressively (provider → shortcuts → menu → styles)
5. Test that existing features still work (drag, connect, deploy, save)
6. Commit: `git add -A && git commit -m "feat(ui): integrate @ant-design/pro-flow for enhanced canvas"`
