# Task: Port Argo OperationsMap and Utility Components

Port key Argo Workflows UI components into our DataBrew pipeline frontend.

## Source Code
Argo UI source is at `/Users/rick/src/argo-workflows/ui/src/`
Our frontend is at `/Users/rick/cyber-databrew/frontend/src/`

## Key Files in Argo Source

### 1. WorkflowOperationsMap
- Path: `shared/workflow-operations-map.ts`
- This is a declarative map defining 7 operations: RETRY, RESUBMIT, SUSPEND, RESUME, STOP, TERMINATE, DELETE
- Each operation has: `{ title, action, iconClassName, disabled(wf) }`
- This is reused for BOTH the batch toolbar AND the detail page action menu
- Port this as a utility that returns Ant Design Button props

### 2. Phase/PhaseIcon
- Path: looks for files with "phase" in name
- Status color mapping + icon assignment
- We already have `PHASE_COLORS` and `STATUS_COLORS` in WorkflowDetailPage.tsx — consolidate into a shared utility

### 3. WorkflowLabels
- Label rendering for workflow list
- Render as Ant Design Tag components

### 4. DurationPanel
- Duration formatting from startedAt/finishedAt
- Progress bar for running workflows

### 5. LinkifiedText
- URL auto-detection in text

## What to Build

### New utility: `Frontend/src/lib/workflow-operations.ts`
A declarative operations map that returns button configs based on workflow phase:

```typescript
const WORKFLOW_OPERATIONS = {
  stop: { title: '停止', icon: <PauseCircleOutlined />, phases: ['Running', 'Pending'], action: ... },
  retry: { title: '重试', icon: <ReloadOutlined />, phases: ['Failed', 'Error'], action: ... },
  resume: { title: '恢复', icon: <PlayCircleOutlined />, phases: ['Suspended'], action: ... },
  suspend: { title: '暂停', icon: <PauseCircleFilled />, phases: ['Running'], action: ... },
  terminate: { title: '终止', icon: <StopOutlined />, phases: ['Running', 'Pending'], action: ... },
  resubmit: { title: '重提交', icon: <RedoOutlined />, phases: ['Succeeded', 'Failed', 'Error'], action: ... },
  delete: { title: '删除', icon: <DeleteOutlined />, phases: ['*'], action: ... },
}
```

### Integrate into WorkflowDetailPage
- Add operation buttons to the detail page header (right side, next to view toggle)
- Buttons should be disabled/enabled based on current workflow phase
- Each button calls the corresponding API endpoint

### Integrate into WorkflowListPage
- Add a "操作" dropdown to each row
- Show relevant operations based on workflow status

### New shared utilities
- Move `PHASE_COLORS`, `STATUS_COLORS` from WorkflowDetailPage.tsx to a shared `lib/constants.ts`
- Add `formatDuration(startedAt, finishedAt)` utility
- Add `WorkflowLabels` component that renders label tags

## Steps
1. Read Argo source files for OperationsMap
2. Create `Frontend/src/lib/workflow-operations.ts`
3. Integrate into WorkflowDetailPage.tsx header
4. Integrate into WorkflowListPage.tsx row actions
5. Create shared phase constants
6. Run `npm run lint`
7. Commit: `git add -A && git commit -m "feat(ui): add workflow operations map and shared utilities"`
