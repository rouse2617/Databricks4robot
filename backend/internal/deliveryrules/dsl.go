package deliveryrules

import (
	"encoding/json"
	"fmt"
)

// QueryDSL is the delivery_rules.query_dsl document (v1).
type QueryDSL struct {
	Where      []Predicate `json:"where"`
	AssetTypes []string    `json:"asset_types,omitempty"`
}

type Predicate struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}

func ParseQueryDSL(raw json.RawMessage) (*QueryDSL, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("query_dsl is empty")
	}
	var dsl QueryDSL
	if err := json.Unmarshal(raw, &dsl); err != nil {
		return nil, fmt.Errorf("invalid query_dsl: %w", err)
	}
	if len(dsl.Where) == 0 {
		return nil, fmt.Errorf("query_dsl.where must have at least one predicate")
	}
	for i, p := range dsl.Where {
		if p.Field == "" || p.Op == "" {
			return nil, fmt.Errorf("predicate %d: field and op are required", i)
		}
	}
	return &dsl, nil
}
