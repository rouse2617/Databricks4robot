package openlineage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestBuilderBuildsOpenLineageEvent(t *testing.T) {
	occurredAt := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	got, ok, err := (Builder{
		Namespace: "cyber-databrew-dev",
		Producer:  "https://github.com/CyberOrigin2077/cyber-databrew",
	}).Build(models.AssetEvent{
		EventID:      "event-1",
		EventSeq:     42,
		EventType:    "algo_finished",
		AssetID:      "asset001",
		TenantID:     "tenant-a",
		ProjectID:    "project-a",
		EventPayload: json.RawMessage(`{"algo_key":"hand_tracking@1.2.0","run_id":"run-123"}`),
		OccurredAt:   occurredAt,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !ok {
		t.Fatalf("expected event to be supported")
	}
	if got.EventType != "COMPLETE" {
		t.Fatalf("unexpected event type: %s", got.EventType)
	}
	if got.Run.RunID != "run-123" {
		t.Fatalf("unexpected run id: %s", got.Run.RunID)
	}
	if got.Job.Namespace != "cyber-databrew-dev.jobs" || got.Job.Name != "hand_tracking@1.2.0" {
		t.Fatalf("unexpected job: %+v", got.Job)
	}
	if len(got.Outputs) != 1 || got.Outputs[0].Namespace != "cyber-databrew-dev.assets" || got.Outputs[0].Name != "asset001" {
		t.Fatalf("unexpected outputs: %+v", got.Outputs)
	}
	if got.EventTime != occurredAt {
		t.Fatalf("unexpected event time: %s", got.EventTime)
	}
}

func TestBuilderSkipsUnsupportedEvent(t *testing.T) {
	got, ok, err := (Builder{}).Build(models.AssetEvent{EventType: "asset_created"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if ok || got != nil {
		t.Fatalf("expected event without asset_id to be skipped, got ok=%v event=%+v", ok, got)
	}
}

func TestEmitterPostsJSONAndFailsOnNon2xx(t *testing.T) {
	var received Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	emitter, err := NewEmitter(srv.URL, time.Second)
	if err != nil {
		t.Fatalf("NewEmitter: %v", err)
	}
	err = emitter.Emit(context.Background(), &Event{
		EventType: "COMPLETE",
		EventTime: time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC),
		Producer:  "test",
		SchemaURL: schemaURL,
		Run:       Run{RunID: "run-1"},
		Job:       Job{Namespace: "jobs", Name: "job"},
	})
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if received.Run.RunID != "run-1" {
		t.Fatalf("unexpected received event: %+v", received)
	}

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer failSrv.Close()
	emitter.Endpoint = failSrv.URL
	if err := emitter.Emit(context.Background(), &received); err == nil {
		t.Fatalf("expected non-2xx error")
	}
}
