package queryir

// QueryRequest is the canonical v1 query request envelope.
type QueryRequest struct {
	SchemaVersion string       `json:"schema_version" binding:"required"`
	Mode          string       `json:"mode,omitempty"`
	Scope         QueryScope   `json:"scope"`
	Select        QuerySelect  `json:"select"`
	Where         *QueryExpr   `json:"where,omitempty"`
	Sort          []QuerySort  `json:"sort,omitempty"`
	Page          QueryPage    `json:"page"`
	Facets        []QueryFacet `json:"facets,omitempty"`
	Debug         QueryDebug   `json:"debug"`
}

type QueryScope struct {
	Resource       string `json:"resource"`
	IncludeHistory bool   `json:"include_history,omitempty"`
}

type QuerySelect struct {
	Fields []string `json:"fields,omitempty"`
}

type QuerySort struct {
	Field     string `json:"field"`
	Direction string `json:"direction,omitempty"`
}

type QueryPage struct {
	// Current bridge paging (public, stable): page/page_size
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	// Target paging (optional, bridge-compatible): offset/limit
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type QueryFacet struct {
	Field string `json:"field"`
	Size  int    `json:"size,omitempty"`
}

type QueryDebug struct {
	Explain bool `json:"explain,omitempty"`
}

type QueryExpr struct {
	And  []QueryExpr     `json:"and,omitempty"`
	Or   []QueryExpr     `json:"or,omitempty"`
	Not  *QueryExpr      `json:"not,omitempty"`
	Pred *QueryPredicate `json:"pred,omitempty"`
}

type QueryPredicate struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}

type FieldCapabilityBrief struct {
	Field   string   `json:"field"`
	Engines []string `json:"engines"`
}

type DebugPlan struct {
	Steps []DebugPlanStep `json:"steps"`
}

type DebugPlanStep struct {
	Engine string `json:"engine"`
	Mode   string `json:"mode"`
}

// CompiledQuery is a v1 bridge to the existing asset filter path.
type CompiledQuery struct {
	FilterStrings     []string
	SortBy            string
	Page              int
	PageSize          int
	NormalizedQuery   QueryRequest
	FieldCapabilities []FieldCapabilityBrief
	DebugPlan         DebugPlan
	Warnings          []string
	Facets            map[string][]FacetBucket
	CandidateAssetIDs []string
	// MatchTotal is set from Elasticsearch track_total_hits when PG COUNT is skipped.
	MatchTotal int64 `json:"match_total,omitempty"`
	// ESResults carries full ES _source data for the first page when
	// ES recall matched (candidate transfer was skipped due to >10k matches).
	// When present, the handler should skip PG refine and return these directly.
	ESResults []map[string]any `json:"-"`
}

type FacetBucket struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// ResultColumn describes one returned column. Used by QueryRunResponse.
type ResultColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}
