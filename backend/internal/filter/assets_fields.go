package filter

import (
	"fmt"
	"regexp"
	"strings"
)

type fieldSpec struct {
	Canonical    string
	StorageField string
	IsJSONB      bool
}

type fieldPrefixSpec struct {
	InputPrefixes   []string
	CanonicalPrefix string
	StoragePrefix   string
}

var dynamicFieldKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_@:/.\-]+$`)

var allowedLifecycleKeys = map[string]bool{
	"retention_tier":     true,
	"archive_after_days": true,
	"delete_after_days":  true,
	"total_size_bytes":   true,
	"last_accessed_at":   true,
}

var exactFieldSpecs = map[string]fieldSpec{
	"asset_id":           {Canonical: "asset_id", StorageField: "asset_id", IsJSONB: false},
	"mcap_file_id":       {Canonical: "mcap_file_id", StorageField: "mcap_file_id", IsJSONB: false},
	"start_timestamp_ns": {Canonical: "start_timestamp_ns", StorageField: "start_timestamp_ns", IsJSONB: false},
	"status":             {Canonical: "status", StorageField: "status", IsJSONB: false},
	"created_at":         {Canonical: "created_at", StorageField: "created_at", IsJSONB: false},
	"updated_at":         {Canonical: "updated_at", StorageField: "updated_at", IsJSONB: false},
	"version":            {Canonical: "version", StorageField: "version", IsJSONB: false},

	"end_timestamp_ns":  {Canonical: "end_timestamp_ns", StorageField: "end_timestamp_ns", IsJSONB: false},
	"duration_sec":      {Canonical: "duration_sec", StorageField: "duration_ms", IsJSONB: false},
	"reviewer":          {Canonical: "reviewer", StorageField: "reviewer", IsJSONB: false},
	"owner":             {Canonical: "owner", StorageField: "owner", IsJSONB: false},
	"type":              {Canonical: "type", StorageField: "asset_type", IsJSONB: false},
	"env":               {Canonical: "env", StorageField: "cf_meta.env", IsJSONB: true},
	"task":              {Canonical: "task", StorageField: "cf_meta.task", IsJSONB: true},
	"delivery_count":    {Canonical: "delivery_count", StorageField: "delivery_count", IsJSONB: false},
	"last_delivered_to": {Canonical: "last_delivered_to", StorageField: "last_delivered_to", IsJSONB: false},
	"last_delivered_at": {Canonical: "last_delivered_at", StorageField: "last_delivered_at", IsJSONB: false},

	// New promoted fields (schema evolution v2)
	"lifecycle_state": {Canonical: "lifecycle_state", StorageField: "lifecycle_state", IsJSONB: false},
	"asset_type":      {Canonical: "asset_type", StorageField: "asset_type", IsJSONB: false},
	"expire_at":       {Canonical: "expire_at", StorageField: "expire_at", IsJSONB: false},

	"retention_tier":     {Canonical: "retention_tier", StorageField: "retention_tier", IsJSONB: false},
	"archive_after_days": {Canonical: "archive_after_days", StorageField: "cf_meta.archive_after_days", IsJSONB: true},
	"delete_after_days":  {Canonical: "delete_after_days", StorageField: "cf_meta.delete_after_days", IsJSONB: true},
	"total_size_bytes":   {Canonical: "total_size_bytes", StorageField: "cf_meta.total_size_bytes", IsJSONB: true},
	"last_accessed_at":   {Canonical: "last_accessed_at", StorageField: "cf_meta.last_accessed_at", IsJSONB: true},
}

var prefixedFieldSpecs = []fieldPrefixSpec{
	{
		InputPrefixes:   []string{"tag.", "tags.", "cf_tag."},
		CanonicalPrefix: "tag.",
		StoragePrefix:   "cf_tag.",
	},
	{
		InputPrefixes:   []string{"algo.", "algo_results.", "cf_algo."},
		CanonicalPrefix: "algo.",
		StoragePrefix:   "cf_algo.",
	},
	{
		InputPrefixes:   []string{"files.", "cf_files."},
		CanonicalPrefix: "files.",
		StoragePrefix:   "cf_files.",
	},
	{
		InputPrefixes:   []string{"lifecycle.", "lifecycle_meta."},
		CanonicalPrefix: "lifecycle.",
		StoragePrefix:   "cf_meta.",
	},
}

func ResolveField(field string) (fieldSpec, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return fieldSpec{}, fmt.Errorf("filter: empty field name")
	}

	if spec, ok := exactFieldSpecs[field]; ok {
		return spec, nil
	}

	for _, prefixSpec := range prefixedFieldSpecs {
		for _, inputPrefix := range prefixSpec.InputPrefixes {
			if !strings.HasPrefix(field, inputPrefix) {
				continue
			}

			key := strings.TrimPrefix(field, inputPrefix)
			if key == "" {
				return fieldSpec{}, fmt.Errorf("filter: field %q is missing a key suffix", field)
			}
			if !dynamicFieldKeyPattern.MatchString(key) {
				return fieldSpec{}, fmt.Errorf("filter: field %q contains unsupported characters", field)
			}
			if prefixSpec.CanonicalPrefix == "lifecycle." && !allowedLifecycleKeys[key] {
				return fieldSpec{}, fmt.Errorf("filter: field %q is not allowed", field)
			}

			return fieldSpec{
				Canonical:    prefixSpec.CanonicalPrefix + key,
				StorageField: prefixSpec.StoragePrefix + key,
				IsJSONB:      true,
			}, nil
		}
	}

	return fieldSpec{}, fmt.Errorf("filter: field %q is not allowed", field)
}
