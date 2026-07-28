package models

// CYB-4305: batch asset lineage lookup — request/response types for
// POST /api/v1/assets/lineage-batch. See openspec/changes/CYB-4305-lineage.
//
// Kept in a dedicated file (not asset.go) so the batch-lookup surface stays
// organized per feature — matches the durations/costs layout.

// AssetLineageBatchRequest is the body accepted by POST /api/v1/assets/lineage-batch.
//
// `ids` may contain any mix of asset_id (8-char) and grace_video_id (uuid);
// `id_type` is a UI hint — the server always matches both columns. `depth`
// accepts both `1` (JSON number), `"1"` (string) and `"all"` — the handler
// normalizes to a bool.
type AssetLineageBatchRequest struct {
	IDs    []string `json:"ids"`
	IDType string   `json:"id_type,omitempty"`
	// Depth is `any` on the wire so JSON `1`, `"1"`, and `"all"` are all
	// accepted; the handler normalizes and validates before calling the usecase.
	Depth any `json:"depth,omitempty"`
}

// LineageBatchItem is one row of the response `items` array.
//
// Parent/root/logical are pointers so a NULL DB column serializes as JSON
// `null` (not `""`), letting callers distinguish "unknown parent" from
// "known empty parent". The depth=all-only fields (`upstream_ids`,
// `downstream_ids`, `relation_types`) are pointers to slices for the same
// reason plus a size win — depth=1 responses omit them entirely rather
// than emitting `[]`.
type LineageBatchItem struct {
	InputID        string  `json:"input_id"`
	AssetID        string  `json:"asset_id"`
	GraceVideoID   string  `json:"grace_video_id,omitempty"`
	ParentAssetID  *string `json:"parent_asset_id"`
	RootAssetID    *string `json:"root_asset_id"`
	LogicalAssetID *string `json:"logical_asset_id"`
	IsCurrent      bool    `json:"is_current"`
	Revision       int64   `json:"revision"`

	// Depth=all-only fields. When depth=1 these are nil and omitempty drops
	// them from the payload; when depth=all they may be empty slices (e.g.
	// for a leaf asset with no downstream).
	UpstreamIDs   *[]string `json:"upstream_ids,omitempty"`
	DownstreamIDs *[]string `json:"downstream_ids,omitempty"`
	RelationTypes *[]string `json:"relation_types,omitempty"`
}

// LineageBatchStats are computed server-side over the response `items` array
// (matched assets only). RelationTypeCounts is present when depth="all"
// (possibly empty when no items carry relation types) and absent when
// depth=1 (usecase passes nil so the omitempty tag on the *pointer* field
// drops it). A plain `map[string]int` with omitempty would drop an empty
// map too, which is the wrong shape for callers that render the histogram
// unconditionally.
type LineageBatchStats struct {
	MatchedCount       int             `json:"matched_count"`
	MissingCount       int             `json:"missing_count"`
	HasParentCount     int             `json:"has_parent_count"`
	IsRootCount        int             `json:"is_root_count"`
	OrphanCount        int             `json:"orphan_count"`
	IsCurrentCount     int             `json:"is_current_count"`
	RelationTypeCounts *map[string]int `json:"relation_type_counts,omitempty"`
}

// AssetLineageBatchResponse is the body returned by POST /api/v1/assets/lineage-batch.
//
// FilteredOutIDs is kept in the response envelope for symmetry with the
// costs/durations shells (frontend BatchAssetLookup reads it uniformly);
// lineage has no range filter so this is always empty.
type AssetLineageBatchResponse struct {
	Items          []LineageBatchItem `json:"items"`
	MissingIDs     []string           `json:"missing_ids"`
	FilteredOutIDs []string           `json:"filtered_out_ids"`
	Stats          LineageBatchStats  `json:"stats"`
}
