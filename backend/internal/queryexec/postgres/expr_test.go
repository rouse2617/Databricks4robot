package postgres

import (
	"strings"
	"testing"

	"data-platform/internal/queryir"
)

func TestBuildExprWhereClause_OrAndNot(t *testing.T) {
	expr := &queryir.QueryExpr{
		Or: []queryir.QueryExpr{
			{Pred: &queryir.QueryPredicate{Field: "owner", Op: "eq", Value: "alice"}},
			{Not: &queryir.QueryExpr{
				Pred: &queryir.QueryPredicate{Field: "lifecycle_state", Op: "eq", Value: "archived"},
			}},
		},
	}

	clause, nextParam, err := BuildExprWhereClause(expr, 1)
	if err != nil {
		t.Fatalf("BuildExprWhereClause() err = %v", err)
	}
	if nextParam != 3 {
		t.Fatalf("unexpected nextParam: %d", nextParam)
	}
	if len(clause.Args) != 2 {
		t.Fatalf("unexpected args: %#v", clause.Args)
	}
	if !strings.Contains(clause.SQL, "owner = $1") {
		t.Fatalf("expected owner predicate in SQL, got %q", clause.SQL)
	}
	if !strings.Contains(clause.SQL, "NOT (lifecycle_state = $2)") {
		t.Fatalf("expected NOT predicate in SQL, got %q", clause.SQL)
	}
	if !strings.Contains(clause.SQL, " OR ") {
		t.Fatalf("expected OR in SQL, got %q", clause.SQL)
	}
}

func TestBuildCandidateIDsClause(t *testing.T) {
	sql, args := buildCandidateIDsClause([]string{"aset0001", "aset0002"}, 3)
	if sql != "asset_id IN ($3, $4)" {
		t.Fatalf("unexpected sql: %q", sql)
	}
	if len(args) != 2 || args[0] != "aset0001" || args[1] != "aset0002" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildExprWhereClause_FulltextIncludesIDs(t *testing.T) {
	expr := &queryir.QueryExpr{
		Pred: &queryir.QueryPredicate{Field: "_fulltext", Op: "ilike", Value: "2GV9eFCT"},
	}
	clause, nextParam, err := BuildExprWhereClause(expr, 1)
	if err != nil {
		t.Fatalf("BuildExprWhereClause() err = %v", err)
	}
	if nextParam != 2 {
		t.Fatalf("unexpected nextParam: %d", nextParam)
	}
	if len(clause.Args) != 1 || clause.Args[0] != "%2GV9eFCT%" {
		t.Fatalf("unexpected args: %#v", clause.Args)
	}
	if !strings.Contains(clause.SQL, "assets.asset_id::text ILIKE $1") {
		t.Fatalf("expected asset_id fulltext in SQL, got %q", clause.SQL)
	}
	if !strings.Contains(clause.SQL, "assets.mcap_file_id::text ILIKE $1") {
		t.Fatalf("expected mcap_file_id fulltext in SQL, got %q", clause.SQL)
	}
}
