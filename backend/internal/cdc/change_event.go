package cdc

import "fmt"

// Operation mirrors common CDC operation codes used by Debezium-style events.
type Operation string

const (
	OperationCreate Operation = "c"
	OperationUpdate Operation = "u"
	OperationDelete Operation = "d"
	OperationRead   Operation = "r"
)

// ChangeEvent is the normalized shape the application-level CDC consumers use.
// It is transport-agnostic on purpose: a Debezium decoder, a managed CDC
// bridge, or tests can all map into this type.
type ChangeEvent struct {
	Table  string
	Op     Operation
	Key    map[string]any
	Before map[string]any
	After  map[string]any
}

// StringField returns the first non-empty string value found under key across
// After, Before and Key, in that order.
func (e ChangeEvent) StringField(key string) string {
	for _, source := range []map[string]any{e.After, e.Before, e.Key} {
		if source == nil {
			continue
		}
		if raw, ok := source[key]; ok && raw != nil {
			return fmt.Sprint(raw)
		}
	}
	return ""
}
