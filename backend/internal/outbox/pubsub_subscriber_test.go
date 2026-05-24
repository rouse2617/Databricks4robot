package outbox

import (
	"context"
	"errors"
	"testing"
)

type fakePubSubMessage struct {
	acked  int
	nacked int
}

func (m *fakePubSubMessage) Ack()  { m.acked++ }
func (m *fakePubSubMessage) Nack() { m.nacked++ }

func TestValidatePubSubEventSourceConfig(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		subName   string
		wantErr   bool
	}{
		{name: "valid", projectID: "project", subName: "asset-events", wantErr: false},
		{name: "missing project", projectID: "", subName: "asset-events", wantErr: true},
		{name: "missing subscription", projectID: "project", subName: " ", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePubSubEventSourceConfig(tt.projectID, tt.subName)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPubSubEventMessageAckOnSuccess(t *testing.T) {
	msg := &fakePubSubMessage{}
	var got []byte
	handlePubSubEventMessage(context.Background(), []byte(`{"event_seq":1}`), msg, func(_ context.Context, data []byte) error {
		got = data
		return nil
	})

	if string(got) != `{"event_seq":1}` {
		t.Fatalf("handler got payload %s", string(got))
	}
	if msg.acked != 1 || msg.nacked != 0 {
		t.Fatalf("expected ack=1 nack=0, got ack=%d nack=%d", msg.acked, msg.nacked)
	}
}

func TestPubSubEventMessageNackOnHandlerError(t *testing.T) {
	msg := &fakePubSubMessage{}
	handlePubSubEventMessage(context.Background(), []byte(`{"event_seq":1}`), msg, func(context.Context, []byte) error {
		return errors.New("index failed")
	})

	if msg.acked != 0 || msg.nacked != 1 {
		t.Fatalf("expected ack=0 nack=1, got ack=%d nack=%d", msg.acked, msg.nacked)
	}
}

func TestPubSubSubscriberReceiveRejectsIncompleteWiring(t *testing.T) {
	if err := (*PubSubSubscriber)(nil).Receive(context.Background(), func(context.Context, []byte) error { return nil }); err == nil {
		t.Fatalf("expected nil subscriber wiring error")
	}
	sub := &PubSubSubscriber{}
	if err := sub.Receive(context.Background(), func(context.Context, []byte) error { return nil }); err == nil {
		t.Fatalf("expected incomplete subscriber wiring error")
	}
}
