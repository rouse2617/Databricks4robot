# Spec — Asset metadata endpoint (CYB-3203)

## MODIFIED: asset-metadata (new capability)

### New endpoint
`GET /api/v1/assets/:id/metadata`

### Response (200)
```json
{
  "asset_id": "...",
  "segment_locator": "...",
  "lifecycle_state": "...",
  "asset_metadata": { ... },          // raw assets.metadata JSONB
  "mcap_metadata": { ... },            // raw mcap_files.metadata JSONB, or null
  "grace_video_snapshot": { ... }|null, // = asset_metadata.grace_video_snapshot
  "storage": { "gcs": {...}, "aliyun": {...} }|null, // = mcap_metadata.storage_meta
  "collection": { ... }|null,         // = mcap_metadata.collection_meta
  "process_info": { ... }|null,       // = mcap_metadata.process_info
  "video_info": { ... }|null          // = mcap_metadata.video_info
}
```

### Errors
- `404` — asset not found
- `401` — no/invalid auth
- `200` with `mcap_metadata: null` when asset exists but has no mcap_files row
  (grace_video without mcap); all convenience fields are also null in this case.

### Auth
- Same as `GET /api/v1/assets/:id` — any `Authenticate`-passing principal;
  no additional `RequireScope` guard.

### UI: "Advanced / metadata" collapsible section (UX-1.5)

> **Single Antd `<Collapse>` panel wrapping one `<JsonViewer>` component.**
> No tabs. Default collapsed. When expanded, the JSON tree shows **level 1 only**
> (i.e. just `files` / `storage` / `processing` / `asset_metadata` / `mcap_metadata`
> node names + item counts). Users click into any subtree to drill down.

Component sketch:
```tsx
<Collapse ghost>
  <Collapse.Panel header="高级 / 元数据" key="meta">
    {loading
      ? <Spin />
      : <JsonViewer
          value={data}
          collapsed={1}        // expand only level 1
          enableClipboard
          displayDataTypes={false}
        />
    }
  </Collapse.Panel>
</Collapse>
```

UX notes:
- The level-1 keys are the user's navigation handles — they should match the
  natural mental model (e.g. `asset_metadata`, `mcap_metadata`, and a small set
  of flattened convenience subtrees for grace_video / storage / processing /
  collection / video_info).
- Frontend is **read-only** — no inline editor, no copy-to-clipboard surprises
  beyond standard `enableClipboard` (user-initiated).
- The convenience fields are **redundant** with the raw `*_metadata` trees; the
  UI does not need to display them separately. The **raw trees are the single
  source of truth** rendered in the viewer. The convenience fields exist so
  *if* the UI later wants to show, e.g., "Storage" as a small card, it can
  without re-parse. **v1 just hands both to the JsonViewer**.

### Frontend dependency
- **`react-json-view`** (or equivalent tree viewer with `collapsed` prop). If
  not already in `Frontend/package.json`, add it as a runtime dep and declare
  in the PR description (AI-RULES dependency-declaration rule).
- If the team prefers zero-dep, a ~30-line recursive `<JsonNode>` component
  (TypeScript, no external dep) is an acceptable substitute — same UX
  (collapsed={1} default), just a few more lines of code.

### Non-goals
- Edit / mutate metadata via API.
- Caching / ETag.
- Truncation of large JSONB (v1; revisit if sizes grow).
