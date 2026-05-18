package outbox

import (
	"context"
	"testing"
)

func TestPublisherNilTopic(t *testing.T) {
	p := &Publisher{}
	if _, err := p.Publish(context.Background(), "a1", []byte(`{}`)); err == nil {
		t.Fatalf("expected error for nil topic")
	}
}

func TestPublisherResumeAndCloseNilSafe(t *testing.T) {
	var p *Publisher
	p.ResumePublishAfterError("")
	if err := p.Close(); err != nil {
		t.Fatalf("nil close should be safe: %v", err)
	}
}
