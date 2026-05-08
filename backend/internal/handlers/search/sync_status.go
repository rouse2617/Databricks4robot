package search

// SyncInfo describes how the search index is kept in sync with PostgreSQL
// (CDC vs dev reconciler vs manual), for UI and demo observability.
type SyncInfo struct {
	ElasticsearchOK bool   `json:"elasticsearch_ok"`
	CDCEnabled      bool   `json:"cdc_enabled"`
	CDCSourceDriver string `json:"cdc_source_driver,omitempty"`
	// SearchIndexMode is one of: unavailable | cdc | local_reconcile | manual
	SearchIndexMode string `json:"search_index_mode"`
	Env             string `json:"env,omitempty"`
}
