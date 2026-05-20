## ADDED Requirements

### Requirement: Preview manifest drives video source selection
The system SHALL load mcap-preview manifest when opening asset quick preview or asset detail preview and expose `sources` for topic selection.

**Priority**: P0 (Critical)
**Rationale**: Without manifest, users cannot discover or switch CompressedVideo topics on list or detail surfaces.

#### Scenario: Multi-topic asset shows source picker on list sidebar
- **Given** an asset with multiple `foxglove.CompressedVideo` channels
- **When** the user opens quick preview on the assets workbench
- **Then** the preview pane lists manifest `sources` with topic and codec labels
- **And** the user can search/filter sources by label
- **And** changing the source updates `segment.mp4` URL including `topic`

#### Scenario: Multi-topic asset shows source picker on asset detail
- **Given** an asset with multiple `foxglove.CompressedVideo` channels
- **When** the user opens the asset detail page
- **Then** the preview hero lists manifest `sources` with topic and codec labels
- **And** the user can search/filter sources by label
- **And** changing the source updates `segment.mp4` URL including `topic`
- **And** the selected source id is reflected in the URL query `preview_source`

#### Scenario: Preview source ids are stable across manifest requests
- **Given** an asset with multiple `foxglove.CompressedVideo` channels
- **When** the client calls `GET /preview/assets/:id/manifest` repeatedly
- **Then** each manifest `sources[].id` maps to the same `sources[].topic` on every response
- **And** ids are assigned in deterministic topic order (`live_topic_0`, `live_topic_1`, … without gaps from skipped channels)

#### Scenario: Direct link with preview_source opens the matching topic
- **Given** a valid manifest source id in `?preview_source=`
- **When** the user opens the asset detail page with that query string
- **Then** the preview hero selects that source
- **And** `segment.mp4` is requested with the matching `topic` query parameter

#### Scenario: Invalid preview_source is corrected
- **Given** `?preview_source=` names an id not present in the current manifest
- **When** the asset detail page loads preview
- **Then** the UI selects the recommended or first valid source
- **And** the URL is updated to that valid id without an infinite reload loop
