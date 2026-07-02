# Context files — CYB-1613

backend/internal/handlers/workflow/handler.go # workflow detail response generation
backend/internal/handlers/workflow/dag_edges.go # DAG edge construction
backend/internal/handlers/workflow/handler_test.go # workflow detail and edge tests
backend/internal/handlers/workflow/pod_name.go # pod resolution for runtime nodes
Frontend/src/pages/WorkflowDetailPage.tsx # consumer of workflow detail nodes/edges
Frontend/src/pages/WorkflowDagView.tsx # graph rendering assumptions
Frontend/src/pages/WorkflowDagNode.tsx # node phase display assumptions
api/openapi.yaml # workflow detail response contract reference
docs/review/api-guide.md # workflow detail docs if contract wording changes
