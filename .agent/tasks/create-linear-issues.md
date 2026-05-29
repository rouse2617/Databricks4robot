# Task: Create Linear issues for completed pipeline work

## Background

We've completed two pieces of work on `feat/pipeline-integration`:

### 1. Pipeline Template Save/Load UX (Frontend)
- **What:** Empty pipeline canvas now shows template list ("我的流水线"), `?templateId=xxx` URL param auto-loads template to canvas, deploy dialog shows current template name
- **Files changed:** `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/styles/pipeline.css`
- **Commits:** `e378c83` (feat: pipeline template save/load ux)
- **Labels:** feature

### 2. Workflow Logs podName Fix (Backend)
- **What:** `GetWorkflowLogs` now passes `podName=nodeId` to Argo API so step-level pod logs are fetched correctly
- **Files changed:** `backend/internal/argo/client.go`
- **Commits:** `b790eb6` (fix: pass podname to workflow logs endpoint)
- **Labels:** bug

## Steps

1. Use Linear MCP to check available projects and states in the `cyber-databrew` Linear workspace
2. Search for existing issues related to pipeline template or workflow logs - if none found, create new ones
3. For each completed work item:
   - Create a Linear issue with description, labels, and commit references
   - Set state to `Done`
   - Add a comment with the deploy summary

## Deploy Summary

- **Backend revision:** `cyber-databrew-backend-dev-00355-dk7` (100% traffic)
- **Image tag:** `e378c83`
- **Branch:** `feat/pipeline-integration` (both commits pushed to origin)
