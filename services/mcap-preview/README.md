# mcap-preview

Standalone Go service that will eventually stream MCAP video segments on
demand. This iteration ships the **skeleton** plus the read-only **manifest**
endpoint — no remux, no video bytes yet.

## Architecture

```
client ──HTTP──▶ mcap-preview ──HTTP──▶ cyber-databrew backend
                       │                  (GET /api/v1/assets/:id/mcap-locator)
                       ▼
                     GCS  (Range reads against the MCAP object via a
                           page-cached io.ReadSeeker; see internal/gcsrs/)
```

- HTTP framework: Gin (matches the cyber-databrew backend conventions).
- Auth: pass-through `X-Grace-Token` (phase-0). The token reaches us from the
  caller; we forward it to the upstream backend along with `X-Request-ID`.
- MCAP parsing: only the summary section (`mcap.Reader.Info()`). Topic and
  chunk enumeration is O(channels + chunk_indexes), no message scan.
- GCS IO: `internal/gcsrs` implements an LRU-paged `io.ReadSeeker` so that the
  many small read+seek calls the MCAP reader emits collapse into a handful of
  Range GETs. Pattern is copied from `backend/cmd/mcap-range-demo/`; the two
  modules deliberately stay independent.

## Endpoints

### `GET /healthz`

No auth. Returns `{"status":"ok"}`.

### `GET /api/v1/preview/assets/:id/manifest`

Headers:

- `X-Grace-Token` *(required)*
- `X-Request-ID` *(optional; generated if absent and propagated upstream)*

Behaviour:

1. Validate asset id (`[A-Za-z0-9_-]{1,128}`).
2. Call upstream `GET {UPSTREAM_BASE_URL}/api/v1/assets/{id}/mcap-locator`
   forwarding the token. Upstream `4xx`/`5xx` codes are propagated verbatim
   (status + `code`/`message`/`details`).
3. Open the MCAP object via the page-cached GCS reader.
4. Parse `Info()` and emit:

```json
{
  "asset_id": "...",
  "mcap": { "mcap_uri": "gs://...", "size_bytes": 0 },
  "window": { "start_timestamp_ns": 0, "end_timestamp_ns": 0, "duration_ms": 0 },
  "candidate_video_topics": [
    {
      "topic": "/camera/front",
      "schema_name": "foxglove.CompressedVideo",
      "encoding": "protobuf",
      "message_count_in_window": 0
    }
  ],
  "chunks_in_window": [
    {
      "chunk_index": 12,
      "message_start_time": 1775435036056908229,
      "message_end_time":   1775435056056908229,
      "chunk_start_offset": 16384,
      "chunk_length":       1048576
    }
  ],
  "stats": { "gcs_range_requests": 3, "gcs_bytes_read": 16384 }
}
```

`message_count_in_window` is intentionally `0` in this iteration — gathering
it accurately requires a chunk decode, which is out of scope for the
manifest-only PR.

### `GET /api/v1/preview/assets/:id/segment.mp4`

Streams the asset's MCAP video window as a fragmented MP4 (`Content-Type:
video/mp4`, `Cache-Control: no-store`). The response is sent chunked: an
init segment (ftyp+moov with `avcC` built from in-band SPS/PPS) is flushed
first, followed by media segments (moof+mdat) of roughly one second of frames
each, cut on keyframe boundaries.

Headers:

- `X-Grace-Token` *(canonical)*. **Or** `?grace_token=` query param —
  accepted only on this endpoint as a fallback because the HTML `<video>`
  element cannot send custom headers. Treat the query param like the header
  for security purposes; rotating `GRACE_TOKEN` invalidates both.

Query params:

- `topic` *(optional)*: explicit MCAP topic. Defaults to the first channel
  whose schema name contains `CompressedVideo`.

Behaviour:

1. Validate asset id; resolve `mcap-locator` upstream as in `manifest`.
2. Open the MCAP via the page-cached GCS reader.
3. On the first message, decode the `foxglove.CompressedVideo` payload to
   read its `format` field. Only `h264` is accepted; anything else returns
   `415 UNSUPPORTED_PREVIEW_CODEC`.
4. Extract SPS/PPS from the in-band Annex-B NALUs of the first IDR. If
   neither is present return `422 PREVIEW_NO_SPS_PPS`.
5. Remux subsequent samples into fMP4 fragments (~1s each) and flush after
   each segment. Aborts cleanly on client disconnect.

Error codes specific to this endpoint:

- `415 UNSUPPORTED_PREVIEW_CODEC` — message `format` is not `"h264"`.
- `422 PREVIEW_NO_SPS_PPS` — first IDR sample lacks SPS/PPS NALUs.
- `422 NO_PREVIEW_TOPIC` — no `foxglove.CompressedVideo` channel found.

## Configuration (env)

| Var | Default | Notes |
|---|---|---|
| `PORT` | `8090` | HTTP listen port. |
| `UPSTREAM_BASE_URL` | *(required for manifest)* | e.g. `http://cyber-databrew-backend:8080`. Do **not** include `/api/v1`. |
| `GRACE_TOKEN_PASSTHROUGH` | `true` | When `false`, the upstream call is made anonymously. |
| `GCS_PAGE_SIZE_BYTES` | `1048576` | Page size of the GCS Range reader. |
| `GCS_PAGE_CACHE_BYTES` | `67108864` | Total cache footprint per reader. |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error`. |
| `LOG_FORMAT` | `json` | `json` or `text` (slog handler). |

GCS auth uses Application Default Credentials. In GKE, attach Workload
Identity to the pod's service account and grant `roles/storage.objectViewer`
on the relevant bucket(s).

## Local development

```bash
cd services/mcap-preview
go mod tidy
go build ./...
go test ./...

# Run against a port-forwarded dev backend:
kubectl -n cyber-databrew-dev port-forward svc/cyber-databrew-backend 8080:8080 &
UPSTREAM_BASE_URL=http://localhost:8080 \
  GOOGLE_APPLICATION_CREDENTIALS=$HOME/.config/gcloud/application_default_credentials.json \
  go run ./cmd/server

# Then call the local service:
curl -H "X-Grace-Token: $TOKEN" \
  http://localhost:8090/api/v1/preview/assets/<asset_id>/manifest | jq
```

## Testing

```bash
go test ./...
```

The handler tests stub both the upstream backend (via `httptest.Server`) and
the MCAP source (via `bytes.NewReader` over a synthetic indexed MCAP built
with `mcap.NewWriter`). No GCS access is required.

## Deployment

Kustomize base: `deploy/k8s/mcap-preview/`.

```bash
kubectl apply -n cyber-databrew-dev -k deploy/k8s/mcap-preview/
```

Image: `us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/mcap-preview:dev-latest`.

The same Kustomize base also wires the Gateway API plumbing (HTTPRoute,
GCPBackendPolicy, HealthCheckPolicy, ReferenceGrant), so once applied the
service is reachable behind the existing developer gateway as
`https://api-cyber-databrew-dev.cyberorigin.ai/api/v1/preview/...`. Path
prefix `/api/v1/preview/` is dispatched to `mcap-preview`; everything else
on the same hostname continues to land on `cyber-databrew-backend`.

## Open Questions

- **Mid-stream parameter set changes.** SPS/PPS are captured once at the
  init segment and not re-emitted in fragments. A producer that legitimately
  rotates parameter sets mid-clip would render incorrectly; we have not seen
  this in practice but it deserves a probe before relying on the preview for
  QA.
- **Fragment cadence.** We cut at the next keyframe after ~1s of accumulated
  duration. If the source GOP is longer than ~2s the first fragment will be
  larger and time-to-first-paint will degrade. Capping fragment size or
  forcing periodic IDR upstream would help.
- **Video codec detection.** The skeleton reports `encoding =
  Channel.MessageEncoding` in the manifest (e.g. `protobuf`). The actual
  codec (`h264` / `h265` / `vp9`) lives inside the message payload of
  `foxglove.CompressedVideo` and is now read by the streaming endpoint, but
  the manifest endpoint does not yet sample one message per topic to surface
  it — that's a separate optimization.
- **Window handling when locator window is zero.** We currently fall back to
  `Statistics.MessageStartTime/EndTime`. If the asset row's window is the
  contractually-correct one, we should probably 422 instead — confirm with
  asset owners.
- **HTTPRoute / IAP.** Mirroring `deploy/k8s/gateway/backend-route.yaml` is
  trivial but pulls in cluster-wide DNS / cert decisions; deferring to a
  dedicated PR.
- **Workload Identity SA.** No GCP SA is created yet; pod uses node default
  credentials in dev. Production should bind a least-privilege SA with
  `storage.objectViewer` on the MCAP bucket.
