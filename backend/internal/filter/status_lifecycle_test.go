package filter

import "testing"

func TestBuildStatusCondition_ApprovedMapsToLifecycleStates(t *testing.T) {
	f := Filter{Field: "status", Op: "=", Value: "approved"}
	clause, err := BuildWhereClause([]Filter{f}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause: %v", err)
	}
	want := "lifecycle_state IN ($1, $2, $3, $4)"
	if clause.SQL != want {
		t.Fatalf("SQL: got %q want %q", clause.SQL, want)
	}
	if len(clause.Args) != 4 {
		t.Fatalf("args len: got %d want 4", len(clause.Args))
	}
}

func TestBuildStatusCondition_Rejected(t *testing.T) {
	f := Filter{Field: "status", Op: "=", Value: "rejected"}
	clause, err := BuildWhereClause([]Filter{f}, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause: %v", err)
	}
	if clause.SQL != "lifecycle_state IN ($1, $2)" {
		t.Fatalf("SQL: got %q", clause.SQL)
	}
}
