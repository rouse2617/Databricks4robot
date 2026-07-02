-- Add cost estimation support to pipeline_run_nodes.
-- Cost is computed during Argo status refresh and stored so
-- historical runs retain the cost at time of execution.

ALTER TABLE pipeline_run_nodes ADD COLUMN estimated_cost_usd DECIMAL(16,8);
