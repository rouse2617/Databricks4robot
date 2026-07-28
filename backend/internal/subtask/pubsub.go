package subtask

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	psv1 "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
)

// pullTimeout bounds a single one-shot Pull RPC. The apiv1 Pull returns as soon
// as any messages are available (or the server's short long-poll elapses), so
// this is only a backstop — unlike the high-level Synchronous Receive, it does
// not sit and wait to fill MaxMessages.
const pullTimeout = 20 * time.Second

// ackTimeout bounds the Ack/Nack RPCs, which run after batch creation on a
// context detached from the pull deadline.
const ackTimeout = 10 * time.Second

type PullResult struct {
	// Messages holds one asset group per received Pub/Sub message, preserving
	// message boundaries so the caller can dispatch each message independently
	// (one asset → single run, more → batch). Ack/Nack stay pull-level.
	Messages [][]string
	Ack      func()
	Nack     func()
}

// PubSubClient wraps a single apiv1 SubscriberClient. The subscription resource
// name carries the project, so one client serves every project.
type PubSubClient struct {
	mu     sync.Mutex
	client *psv1.SubscriberClient
}

func NewPubSubClient() *PubSubClient {
	return &PubSubClient{}
}

func (pc *PubSubClient) getClient(ctx context.Context) (*psv1.SubscriberClient, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.client != nil {
		return pc.client, nil
	}
	c, err := psv1.NewSubscriberClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("pubsub subscriber client: %w", err)
	}
	pc.client = c
	return c, nil
}

// Pull does a single synchronous pull of up to maxMessages, returning
// immediately once the server responds. Malformed messages are Acked here so
// they don't redeliver; valid ones are returned with an Ack/Nack that the
// caller invokes after batch creation succeeds/fails.
func (pc *PubSubClient) Pull(ctx context.Context, projectID, subscriptionID string, maxMessages int) (*PullResult, error) {
	client, err := pc.getClient(ctx)
	if err != nil {
		return nil, err
	}

	sub := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionID)

	pullCtx, cancel := context.WithTimeout(ctx, pullTimeout)
	defer cancel()

	resp, err := client.Pull(pullCtx, &pubsubpb.PullRequest{
		Subscription: sub,
		MaxMessages:  int32(maxMessages),
	})
	if err != nil {
		return nil, fmt.Errorf("pubsub pull: %w", err)
	}

	var (
		messages  [][]string
		ackIDs    []string
		badAckIDs []string
	)
	for _, rm := range resp.GetReceivedMessages() {
		ids, topic, parseErr := extractAssetIDs(rm.GetMessage().GetData())
		if parseErr != nil {
			slog.Warn("subtask: bad message, acking to skip", "err", parseErr, "msgID", rm.GetMessage().GetMessageId())
			badAckIDs = append(badAckIDs, rm.GetAckId())
			continue
		}
		if topic != "" {
			// topic is a reserved field (CYB-3801): parsed for future routing/
			// labeling, not acted on yet.
			slog.Debug("subtask: message topic", "topic", topic, "msgID", rm.GetMessage().GetMessageId())
		}
		messages = append(messages, ids)
		ackIDs = append(ackIDs, rm.GetAckId())
	}

	// Ack malformed messages immediately (best-effort) so they don't redeliver.
	if len(badAckIDs) > 0 {
		if err := client.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{Subscription: sub, AckIds: badAckIDs}); err != nil {
			slog.Warn("subtask: ack of bad messages failed", "err", err, "count", len(badAckIDs))
		}
	}

	return &PullResult{
		Messages: messages,
		Ack: func() {
			if len(ackIDs) == 0 {
				return
			}
			ackCtx, c := context.WithTimeout(context.WithoutCancel(ctx), ackTimeout)
			defer c()
			if err := client.Acknowledge(ackCtx, &pubsubpb.AcknowledgeRequest{Subscription: sub, AckIds: ackIDs}); err != nil {
				slog.Warn("subtask: ack failed", "err", err, "count", len(ackIDs))
			}
		},
		Nack: func() {
			if len(ackIDs) == 0 {
				return
			}
			nackCtx, c := context.WithTimeout(context.WithoutCancel(ctx), ackTimeout)
			defer c()
			// Nack == set the ack deadline to 0, forcing immediate redelivery.
			if err := client.ModifyAckDeadline(nackCtx, &pubsubpb.ModifyAckDeadlineRequest{
				Subscription:       sub,
				AckIds:             ackIDs,
				AckDeadlineSeconds: 0,
			}); err != nil {
				slog.Warn("subtask: nack failed", "err", err, "count", len(ackIDs))
			}
		},
	}, nil
}

func (pc *PubSubClient) Close() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.client != nil {
		_ = pc.client.Close()
		pc.client = nil
	}
}

type assetMessage struct {
	AssetIDs []string `json:"asset_ids"`
	// Topic is a reserved field (CYB-3801): future routing/labeling. Parsed and
	// returned for logging, not acted on yet. Unknown JSON keys are ignored, so
	// the message format stays forward-extensible.
	Topic string `json:"topic,omitempty"`
}

// extractAssetIDs parses one message's JSON data. The asset set drives dispatch
// (one asset → single run, more → batch). Returns the trimmed, non-empty asset
// ids plus the reserved topic. Errors when no usable asset id remains.
func extractAssetIDs(data []byte) (ids []string, topic string, err error) {
	var msg assetMessage
	if uerr := json.Unmarshal(data, &msg); uerr != nil {
		return nil, "", fmt.Errorf("unmarshal message: %w", uerr)
	}
	for _, a := range msg.AssetIDs {
		if s := strings.TrimSpace(a); s != "" {
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil, "", fmt.Errorf("no asset_ids in message")
	}
	return ids, msg.Topic, nil
}
