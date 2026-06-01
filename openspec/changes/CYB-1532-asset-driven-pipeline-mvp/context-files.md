# Context files — CYB-1532

`backend/internal/usecase/pipeline/usecase.go`
: Current deploy path validates asset IDs, injects `ASSET_*` env vars, submits Argo workflows to one configured namespace, and saves `pipeline_deployments`.

`backend/internal/handlers/pipeline/handler.go`
: Current HTTP surface for templates, deploy, deployments, retry, stop, resource usage, and pipeline output registration.

`backend/internal/models/pipeline.go`
: Current model only has `PipelineTemplate` and workflow-level `PipelineDeployment`.

`backend/migrations/039_pipeline_tables.sql`
: Current pipeline table definitions. New table work requires explicit approval before editing migrations.

`Frontend/src/pages/PipelinePage.tsx`
: Main pipeline screen with design, saved pipeline, execution, and component tabs.

`Frontend/src/components/pipeline/DeployPanel.tsx`
: Saved template list and run history. Current primary run button submits immediately; asset selection is hidden in a dropdown.

`Frontend/src/components/pipeline/AssetPicker.tsx`
: Asset search and multi-select used by run modals.

`Frontend/src/pages/WorkflowDetailPage.tsx`
: Workflow/node detail surface used for pod, log, and resource inspection.
