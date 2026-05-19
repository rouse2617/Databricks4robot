package filter

import (
	"fmt"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// legacyStatusLifecycleStates maps API legacy status values to lifecycle_state sets.
var legacyStatusLifecycleStates = map[string][]string{
	"approved":   {"created", "processing", "ready", "delivered"},
	"rejected":   {"rejected", "failed"},
	"archived":   {"archived"},
	"superseded": {"superseded"},
}

func lifecycleStatesForLegacyStatus(status string) ([]string, bool) {
	states, ok := legacyStatusLifecycleStates[status]
	if ok {
		return states, true
	}
	ls := models.StatusToLifecycle(status)
	if ls == "created" && status != "" && status != "approved" {
		// Unknown legacy status: treat as exact lifecycle_state if valid.
		if models.IsValidLifecycleState(ls) {
			return []string{ls}, true
		}
		return nil, false
	}
	if models.IsValidLifecycleState(ls) {
		return []string{ls}, true
	}
	return nil, false
}

func buildStatusCondition(f Filter, paramIdx int) (string, []interface{}, int, error) {
	col := "lifecycle_state"
	switch f.Op {
	case "IN", "NOT IN":
		arr, ok := f.Value.([]interface{})
		if !ok {
			arr = []interface{}{f.Value}
		}
		if len(arr) == 0 {
			if f.Op == "IN" {
				return "FALSE", nil, paramIdx, nil
			}
			return "TRUE", nil, paramIdx, nil
		}
		allStates := make([]string, 0)
		for _, v := range arr {
			s, ok := v.(string)
			if !ok {
				return "", nil, paramIdx, fmt.Errorf("filter: status IN value must be string")
			}
			states, ok := lifecycleStatesForLegacyStatus(s)
			if !ok {
				return "", nil, paramIdx, fmt.Errorf("filter: unknown status value %q", s)
			}
			allStates = append(allStates, states...)
		}
		return buildLifecycleStateIn(col, allStates, f.Op, paramIdx)
	case "=":
		s, ok := f.Value.(string)
		if !ok {
			return "", nil, paramIdx, fmt.Errorf("filter: status value must be string")
		}
		states, ok := lifecycleStatesForLegacyStatus(s)
		if !ok {
			return "", nil, paramIdx, fmt.Errorf("filter: unknown status value %q", s)
		}
		return buildLifecycleStateIn(col, states, "IN", paramIdx)
	case "!=":
		s, ok := f.Value.(string)
		if !ok {
			return "", nil, paramIdx, fmt.Errorf("filter: status value must be string")
		}
		states, ok := lifecycleStatesForLegacyStatus(s)
		if !ok {
			return "", nil, paramIdx, fmt.Errorf("filter: unknown status value %q", s)
		}
		return buildLifecycleStateIn(col, states, "NOT IN", paramIdx)
	default:
		return "", nil, paramIdx, fmt.Errorf("filter: operator %q not supported for status (use lifecycle_state)", f.Op)
	}
}

func buildLifecycleStateIn(col string, states []string, op string, paramIdx int) (string, []interface{}, int, error) {
	if len(states) == 0 {
		if op == "IN" {
			return "FALSE", nil, paramIdx, nil
		}
		return "TRUE", nil, paramIdx, nil
	}
	placeholders := make([]string, len(states))
	args := make([]interface{}, len(states))
	for i, st := range states {
		placeholders[i] = fmt.Sprintf("$%d", paramIdx+i)
		args[i] = st
	}
	sql := fmt.Sprintf("%s %s (%s)", col, op, strings.Join(placeholders, ", "))
	return sql, args, paramIdx + len(states), nil
}
