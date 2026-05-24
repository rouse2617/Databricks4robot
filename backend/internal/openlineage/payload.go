package openlineage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

const schemaURL = "https://openlineage.io/spec/2-0-2/OpenLineage.json"

type Event struct {
	EventType string    `json:"eventType"`
	EventTime time.Time `json:"eventTime"`
	Producer  string    `json:"producer"`
	SchemaURL string    `json:"schemaURL"`
	Run       Run       `json:"run"`
	Job       Job       `json:"job"`
	Inputs    []Dataset `json:"inputs,omitempty"`
	Outputs   []Dataset `json:"outputs,omitempty"`
}

type Run struct {
	RunID  string         `json:"runId"`
	Facets map[string]any `json:"facets,omitempty"`
}

type Job struct {
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	Facets    map[string]any `json:"facets,omitempty"`
}

type Dataset struct {
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	Facets    map[string]any `json:"facets,omitempty"`
}

type Builder struct {
	Namespace string
	Producer  string
}

func (b Builder) Build(ev models.AssetEvent) (*Event, bool, error) {
	if strings.TrimSpace(ev.AssetID) == "" {
		return nil, false, nil
	}
	producer := strings.TrimSpace(b.Producer)
	if producer == "" {
		producer = "cyber-databrew"
	}
	namespace := strings.TrimSpace(b.Namespace)
	if namespace == "" {
		namespace = "cyber-databrew"
	}

	payload, err := decodePayload(ev.EventPayload)
	if err != nil {
		return nil, false, err
	}
	runID := firstNonEmpty(
		stringFromMap(payload, "run_id"),
		stringFromMap(payload, "algo_run_id"),
		ev.EventID,
		fmt.Sprintf("asset-event-%d", ev.EventSeq),
	)
	jobName := firstNonEmpty(
		stringFromMap(payload, "algo_key"),
		joinNameVersion(stringFromMap(payload, "algo_name"), stringFromMap(payload, "algo_version")),
		ev.EventType,
		"asset_event",
	)
	out := Dataset{
		Namespace: namespace + ".assets",
		Name:      ev.AssetID,
		Facets: map[string]any{
			"dataSource": map[string]any{"name": "cyber-databrew.asset_events"},
		},
	}
	if ev.EventSeq > 0 {
		out.Facets["version"] = map[string]any{"datasetVersion": fmt.Sprintf("%d", ev.EventSeq)}
	}

	return &Event{
		EventType: mapEventType(ev.EventType),
		EventTime: eventTime(ev),
		Producer:  producer,
		SchemaURL: schemaURL,
		Run: Run{
			RunID: runID,
			Facets: map[string]any{
				"cyberDatabrew": map[string]any{
					"eventId":   ev.EventID,
					"eventSeq":  ev.EventSeq,
					"eventType": ev.EventType,
					"tenantId":  ev.TenantID,
					"projectId": ev.ProjectID,
				},
			},
		},
		Job: Job{
			Namespace: namespace + ".jobs",
			Name:      jobName,
		},
		Outputs: []Dataset{out},
	}, true, nil
}

func decodePayload(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func mapEventType(eventType string) string {
	switch eventType {
	case "algo_started":
		return "START"
	case "algo_finished", "asset_created", "asset_updated", "tag_upserted", "version_promoted":
		return "COMPLETE"
	case "algo_failed", "tag_deleted", "asset_deleted":
		return "ABORT"
	default:
		return "RUNNING"
	}
}

func eventTime(ev models.AssetEvent) time.Time {
	if !ev.OccurredAt.IsZero() {
		return ev.OccurredAt.UTC()
	}
	if !ev.CreatedAt.IsZero() {
		return ev.CreatedAt.UTC()
	}
	return time.Now().UTC()
}

func stringFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func joinNameVersion(name, version string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" {
		return ""
	}
	if version == "" {
		return name
	}
	return name + "@" + version
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
