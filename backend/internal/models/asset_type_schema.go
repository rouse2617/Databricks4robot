package models

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// SchemaRegistry stores code-defined asset_type metadata schemas.
type SchemaRegistry struct {
	schemas map[string]assetTypeSchema
}

type assetTypeSchema struct {
	raw      json.RawMessage
	validate func(map[string]interface{}) error
}

// NewSchemaRegistry returns the default asset type schema registry.
func NewSchemaRegistry() *SchemaRegistry {
	r := &SchemaRegistry{schemas: map[string]assetTypeSchema{}}
	r.register("dataset", datasetSchemaJSON, validateDatasetMetadata)
	r.register("annotation_result", annotationResultSchemaJSON, validateAnnotationResultMetadata)
	return r
}

func (r *SchemaRegistry) register(assetType string, raw json.RawMessage, validate func(map[string]interface{}) error) {
	if r == nil {
		return
	}
	r.schemas[assetType] = assetTypeSchema{raw: append(json.RawMessage(nil), raw...), validate: validate}
}

// Validate validates metadata for an asset_type when a schema is registered.
func (r *SchemaRegistry) Validate(assetType string, metadata map[string]interface{}) error {
	if r == nil {
		return nil
	}
	def, ok := r.schemas[strings.TrimSpace(assetType)]
	if !ok {
		return nil
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	return def.validate(metadata)
}

// GetSchema returns the JSON Schema for assetType, or nil when unregistered.
func (r *SchemaRegistry) GetSchema(assetType string) json.RawMessage {
	if r == nil {
		return nil
	}
	def, ok := r.schemas[strings.TrimSpace(assetType)]
	if !ok {
		return nil
	}
	return append(json.RawMessage(nil), def.raw...)
}

var datasetSchemaJSON = json.RawMessage(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://cyber-databrew.local/schemas/asset-types/dataset.json",
  "title": "dataset asset metadata",
  "type": "object",
  "additionalProperties": true,
  "properties": {
    "format": { "type": "string", "enum": ["parquet", "csv", "image", "lidar", "other"] },
    "record_count": { "type": "integer", "minimum": 0 },
    "size_bytes": { "type": "integer", "minimum": 0 },
    "annotation_status": { "type": "string", "enum": ["raw", "annotated", "validated"] },
    "time_range": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "start": { "type": "string", "format": "date-time" },
        "end": { "type": "string", "format": "date-time" }
      }
    },
    "source": { "type": "string" }
  }
}`)

var annotationResultSchemaJSON = json.RawMessage(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://cyber-databrew.local/schemas/asset-types/annotation_result.json",
  "title": "annotation_result asset metadata",
  "type": "object",
  "additionalProperties": true,
  "properties": {
    "tool": { "type": "string" },
    "schema_version": { "type": "string" },
    "annotators": {
      "type": "array",
      "items": { "type": "string" }
    },
    "quality_score": { "type": "number", "minimum": 0, "maximum": 1 },
    "coverage": { "type": "number", "minimum": 0, "maximum": 1 },
    "artifact_uri": { "type": "string" }
  }
}`)

func validateDatasetMetadata(metadata map[string]interface{}) error {
	if err := optionalStringEnum(metadata, "format", "parquet", "csv", "image", "lidar", "other"); err != nil {
		return err
	}
	if err := optionalNonNegativeInteger(metadata, "record_count"); err != nil {
		return err
	}
	if err := optionalNonNegativeInteger(metadata, "size_bytes"); err != nil {
		return err
	}
	if err := optionalStringEnum(metadata, "annotation_status", "raw", "annotated", "validated"); err != nil {
		return err
	}
	if err := optionalString(metadata, "source"); err != nil {
		return err
	}
	if raw, ok := metadata["time_range"]; ok && raw != nil {
		obj, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("metadata.time_range must be an object")
		}
		start, err := optionalRFC3339(obj, "start")
		if err != nil {
			return fmt.Errorf("metadata.time_range.%w", err)
		}
		end, err := optionalRFC3339(obj, "end")
		if err != nil {
			return fmt.Errorf("metadata.time_range.%w", err)
		}
		if start != nil && end != nil && end.Before(*start) {
			return fmt.Errorf("metadata.time_range.end must be greater than or equal to start")
		}
	}
	return nil
}

func validateAnnotationResultMetadata(metadata map[string]interface{}) error {
	for _, key := range []string{"tool", "schema_version", "artifact_uri"} {
		if err := optionalString(metadata, key); err != nil {
			return err
		}
	}
	if raw, ok := metadata["annotators"]; ok && raw != nil {
		items, ok := raw.([]interface{})
		if !ok {
			return fmt.Errorf("metadata.annotators must be an array")
		}
		for i, item := range items {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("metadata.annotators[%d] must be a string", i)
			}
		}
	}
	for _, key := range []string{"quality_score", "coverage"} {
		if err := optionalUnitNumber(metadata, key); err != nil {
			return err
		}
	}
	return nil
}

func optionalString(metadata map[string]interface{}, key string) error {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	if _, ok := raw.(string); !ok {
		return fmt.Errorf("metadata.%s must be a string", key)
	}
	return nil
}

func optionalStringEnum(metadata map[string]interface{}, key string, allowed ...string) error {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	value, ok := raw.(string)
	if !ok {
		return fmt.Errorf("metadata.%s must be a string", key)
	}
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return fmt.Errorf("metadata.%s must be one of %s", key, strings.Join(allowed, ", "))
}

func optionalNonNegativeInteger(metadata map[string]interface{}, key string) error {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	value, ok := integerValue(raw)
	if !ok {
		return fmt.Errorf("metadata.%s must be an integer", key)
	}
	if value < 0 {
		return fmt.Errorf("metadata.%s must be greater than or equal to 0", key)
	}
	return nil
}

func optionalUnitNumber(metadata map[string]interface{}, key string) error {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil
	}
	value, ok := numberValue(raw)
	if !ok {
		return fmt.Errorf("metadata.%s must be a number", key)
	}
	if value < 0 || value > 1 {
		return fmt.Errorf("metadata.%s must be between 0 and 1", key)
	}
	return nil
}

func optionalRFC3339(metadata map[string]interface{}, key string) (*time.Time, error) {
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return nil, nil
	}
	value, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a string", key)
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must be an RFC3339 timestamp", key)
	}
	return &parsed, nil
}

func integerValue(raw interface{}) (int64, bool) {
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		if math.Trunc(v) == v && v >= math.MinInt64 && v <= math.MaxInt64 {
			return int64(v), true
		}
	case json.Number:
		i, err := v.Int64()
		return i, err == nil
	}
	return 0, false
}

func numberValue(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	}
	return 0, false
}
