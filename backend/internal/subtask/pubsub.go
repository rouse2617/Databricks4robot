package subtask

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

type PullResult struct {
	IDs  []string
	Ack  func()
	Nack func()
}

type PubSubClient struct {
	mu      sync.Mutex
	clients map[string]*pubsub.Client
}

func NewPubSubClient() *PubSubClient {
	return &PubSubClient{clients: make(map[string]*pubsub.Client)}
}

func (pc *PubSubClient) getClient(ctx context.Context, projectID string) (*pubsub.Client, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if c, ok := pc.clients[projectID]; ok {
		return c, nil
	}
	c, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub client for %s: %w", projectID, err)
	}
	pc.clients[projectID] = c
	return c, nil
}

func (pc *PubSubClient) Pull(ctx context.Context, projectID, subscriptionID string, maxMessages int) (*PullResult, error) {
	client, err := pc.getClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	sub := client.Subscription(subscriptionID)
	sub.ReceiveSettings.MaxOutstandingMessages = maxMessages
	sub.ReceiveSettings.Synchronous = true

	pullCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

	var (
		mu   sync.Mutex
		ids  []string
		msgs []*pubsub.Message
	)

	err = sub.Receive(pullCtx, func(_ context.Context, msg *pubsub.Message) {
		mu.Lock()
		defer mu.Unlock()

		id, parseErr := extractAssetID(msg.Data)
		if parseErr != nil {
			slog.Warn("subtask: bad message, acking to skip", "err", parseErr, "msgID", msg.ID)
			msg.Ack()
			return
		}

		ids = append(ids, id)
		msgs = append(msgs, msg)

		if len(ids) >= maxMessages {
			cancel()
		}
	})
	cancel()

	if err != nil && ctx.Err() == nil && len(ids) == 0 {
		return nil, fmt.Errorf("pubsub receive: %w", err)
	}

	return &PullResult{
		IDs: ids,
		Ack: func() {
			for _, m := range msgs {
				m.Ack()
			}
		},
		Nack: func() {
			for _, m := range msgs {
				m.Nack()
			}
		},
	}, nil
}

func (pc *PubSubClient) Close() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for _, c := range pc.clients {
		c.Close()
	}
	pc.clients = make(map[string]*pubsub.Client)
}

type assetMessage struct {
	AssetID string `json:"asset_id"`
}

func extractAssetID(data []byte) (string, error) {
	var msg assetMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", fmt.Errorf("unmarshal message: %w", err)
	}
	if msg.AssetID == "" {
		return "", fmt.Errorf("empty asset_id in message")
	}
	return msg.AssetID, nil
}
