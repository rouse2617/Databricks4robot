# Task: Component Registry Page

Build a full component registry for the DataBrew pipeline platform.

## Background
Currently there are only 2 hardcoded components (Pass Through, Python Script). We need a UI to create, edit, list, and delete custom pipeline components.

## Files to Read First
- `Frontend/src/components/pipeline/ComponentManager.tsx` — existing component management
- `backend/internal/handlers/pipeline_component/handler.go` — existing backend handlers
- `backend/internal/repository/` — find the component repository
- `backend/routes/routes.go` — find component routes

## What to Build

### Backend (check if it already works first)
- Check if `GET/POST/PUT/DELETE /api/v1/pipeline-components` routes exist
- If they exist but are incomplete, fix them. If they don't, build:
  - List all components
  - Create component (name, image, type, command[], args[], env[], description)
  - Update component
  - Delete component

### Frontend — ComponentListPage
- New page: `Frontend/src/pages/ComponentListPage.tsx`
- Route: `/components` (add to router)
- Ant Design Table with columns: Name, Image, Type, Description, CreatedAt, Actions
- "新建组件" button → opens modal
- Edit → populates modal with existing values
- Delete → Popconfirm
- Search bar to filter by name

### Component Form Modal
- Fields: name, image (required), type (select: container/script/resource/suspend), command[], args[], env[] (key-value pairs), description
- Use Ant Design Form with validation
- Reference: visual-argo-workflows modal at `/Users/rick/src/reference/visual-argo-workflows/src/components/modals/template/`

### Sidebar Integration
- Add "组件管理" menu item to the sidebar (left nav)
- Place it near the existing "流水线" and "流水线运行" items

## Tech Stack
- React 18 + TypeScript + Ant Design 5
- Go 1.25 + Gin (backend)

## Steps
1. Read existing code to understand the API shape
2. If backend is incomplete, fix it
3. Build the component list page
4. Build the create/edit modal
5. Add nav menu item
6. Add route
7. Run `go build ./cmd/server && go test ./internal/handlers/pipeline_component/`
8. Run `npm run lint`
9. Commit: `git add -A && git commit -m "feat(ui): add component registry page with CRUD"`
