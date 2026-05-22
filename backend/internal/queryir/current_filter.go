package queryir

// CurrentRevisionOnlySQL is appended to PG asset list queries unless include_history is set.
// Legacy rows with NULL is_current remain visible.
const CurrentRevisionOnlySQL = "(COALESCE(is_current, TRUE) = TRUE)"

// ApplyCurrentOnlyFilter returns true when asset queries should hide non-current revisions.
func ApplyCurrentOnlyFilter(req QueryRequest) bool {
	return req.Scope.Resource == ResourceAssets && !req.Scope.IncludeHistory
}
