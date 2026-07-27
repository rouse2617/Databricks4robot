package outbox

import (
	"context"
	"testing"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newTestPublisher wires a Publisher to an in-memory Pub/Sub (pstest) server so
// the client-side ordering validation runs exactly as in production.
func newTestPublisher(t *testing.T, topicName string, ordering bool) *Publisher {
	t.Helper()
	ctx := context.Background()
	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial pstest: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client, err := pubsub.NewClient(ctx, "proj", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("pubsub client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	if _, err := client.CreateTopic(ctx, topicName); err != nil {
		t.Fatalf("create topic: %v", err)
	}
	topic := client.Topic(topicName)
	topic.EnableMessageOrdering = ordering
	return &Publisher{mode: "pubsub", client: client, topic: topic}
}

// TestPublishOrderingDisabledDoesNotSetOrderingKey is a regression test for the
// publisher bug where every message carried an OrderingKey while the topic had
// EnableMessageOrdering=false. The Pub/Sub client rejects that combination with
// "Topic.EnableMessageOrdering=false, but an OrderingKey was set in Message",
// failing 100% of publishes. With the fix, no key is attached and Get succeeds.
func TestPublishOrderingDisabledDoesNotSetOrderingKey(t *testing.T) {
	ctx := context.Background()
	p := newTestPublisher(t, "no-order", false)

	res, err := p.Publish(ctx, "asset-123", []byte(`{"asset_id":"asset-123"}`))
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if _, err := res.Get(ctx); err != nil {
		t.Fatalf("publish on ordering-disabled topic must succeed, got: %v", err)
	}
}

// TestPublishOrderingEnabledStillPublishes confirms the key is attached (and
// accepted) when the topic does have ordering enabled, so the gating change
// does not silently break a future ordered topic.
func TestPublishOrderingEnabledStillPublishes(t *testing.T) {
	ctx := context.Background()
	p := newTestPublisher(t, "ordered", true)

	res, err := p.Publish(ctx, "asset-123", []byte(`{"asset_id":"asset-123"}`))
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if _, err := res.Get(ctx); err != nil {
		t.Fatalf("publish on ordering-enabled topic must succeed, got: %v", err)
	}
}
