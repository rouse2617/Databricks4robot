# Context files — CYB-2224

backend/internal/transpiler/pipeline.go # Pipeline DSL, RuntimeConfigBinding, VolumeMount
backend/internal/transpiler/transpiler.go # Argo Workflow volumes and node volumeMounts
backend/internal/transpiler/validation.go # pipeline validation patterns
backend/internal/usecase/pipeline/usecase.go # deploy path and node runtime config resolution
backend/internal/usecase/pipeline/usecase_test.go # deploy and manifest test patterns
backend/internal/handlers/pipeline/handler.go # pipeline API request/response mapping
backend/routes/routes.go # API route registration
api/openapi.yaml # HTTP contract sync
docs/review/api-guide.md # API guide sync
scripts/api-guide-smoke.sh # dev contract smoke
Frontend/src/api/pipelineApi.ts # pipeline API client patterns
Frontend/src/components/pipeline/NodeConfigPanel.tsx # node editor runtime input UI
Frontend/src/components/pipeline/PipelineNode.tsx # canvas node runtime input badges
Frontend/src/components/pipeline/types.ts # canvas node data shape
Frontend/src/lib/pipeline-design/canvas-to-dsl.ts # persisted DSL serialization
Frontend/src/lib/pipeline-design/dsl-to-canvas.ts # DSL load path
Frontend/src/lib/pipelineContract.test.ts # DSL round-trip tests
