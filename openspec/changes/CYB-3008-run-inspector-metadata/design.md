# Design - CYB-3008 Run Inspector Metadata

## Data Loading

Extend `useWorkflowDetail` with `runMetadataState`:

- `inputs`: response from `listRunInputs`
- `outputs`: response from `listRunOutputs`
- `runtime`: response from `getRunRuntime`
- `loading`
- `error`

The metadata calls are tied to the resolved Run id. They run with the existing Run ledger refresh, but use `Promise.allSettled` so events, asset nodes, cost, and workflow debug remain usable if one metadata endpoint fails.

## UI

Add a compact unframed section to `WorkflowDetailPage` below the asset-node panel and above the event timeline:

- Runtime summary as Ant Design descriptions.
- Inputs table with type, ref, version, node, mount, target file, source.
- Outputs table with type, node, ref, URI.

The section is hidden for external workflows that have no DataBrew Run.

## Compatibility

Runtime debug fields such as `workflowName` remain visible as runtime references, not product identifiers. Existing node detail, log, terminal, and Pod panels continue using the current workflow debug APIs.
