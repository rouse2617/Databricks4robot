package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

const (
	// Tag events have type like "tag.added", "tag.updated", "tag.deleted".
	eventTypePrefixTag = "tag."
	// deliveryReadyTagPrefix marks tags added by the projector.
	deliveryReadyTagPrefix = "delivery_ready:"
)

// DeliveryEligibilityProjector consumes tag change events and manages
// delivery_ready:* tags based on active delivery rules.
type DeliveryEligibilityProjector struct {
	Subscriber EventSubscriber
	Engine     *deliveryrules.Engine
	Tags       repository.AssetTagRepository
	Assets     repository.AssetRepository
	Customers  repository.CustomerRepository
}

func NewDeliveryEligibilityProjector(
	sub EventSubscriber,
	engine *deliveryrules.Engine,
	tags repository.AssetTagRepository,
	assets repository.AssetRepository,
	customers repository.CustomerRepository,
) *DeliveryEligibilityProjector {
	return &DeliveryEligibilityProjector{
		Subscriber: sub,
		Engine:     engine,
		Tags:       tags,
		Assets:     assets,
		Customers:  customers,
	}
}

// Run blocks until ctx is cancelled.
func (p *DeliveryEligibilityProjector) Run(ctx context.Context) error {
	if p == nil || p.Subscriber == nil || p.Engine == nil || p.Tags == nil || p.Assets == nil || p.Customers == nil {
		return errors.New("delivery eligibility projector: incomplete wiring")
	}
	return p.Subscriber.Receive(ctx, p.handleEvent)
}

func (p *DeliveryEligibilityProjector) handleEvent(ctx context.Context, data []byte) error {
	var ev models.AssetEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}

	// Only process tag change events.
	if !strings.HasPrefix(ev.EventType, eventTypePrefixTag) {
		return nil
	}

	// Guard: ignore delivery_ready tags to avoid feedback loop.
	if ev.EventPayload != nil {
		var payload struct {
			TagKey string `json:"tag_key"`
		}
		if err := json.Unmarshal(ev.EventPayload, &payload); err == nil && strings.HasPrefix(payload.TagKey, deliveryReadyTagPrefix) {
			slog.Debug("delivery eligibility projector: skip self-generated tag event", "tag_key", payload.TagKey)
			return nil
		}
	}

	assetID := ev.AssetID
	if assetID == "" {
		return nil
	}

	// Load asset and all tags.
	asset, err := p.Assets.Get(ctx, assetID)
	if err != nil || asset == nil {
		// Asset may have been deleted; nothing to do.
		if err != nil {
			slog.Debug("delivery eligibility projector: asset load error", "asset_id", assetID, "err", err)
		}
		return nil
	}
	tags, err := p.Tags.ListByAsset(ctx, assetID)
	if err != nil {
		return err
	}

	// Determine which customer ID to use for rule lookup.
	// If the asset is associated with a known customer, use it; otherwise use empty for global rules.
	customerID := ""
	if asset.ProjectID != "" {
		c, err := p.Customers.Get(ctx, asset.ProjectID)
		if err == nil && c != nil {
			customerID = c.CustomerID
		}
	}

	// Evaluate all active rules.
	violations, err := p.Engine.CheckAll(ctx, customerID, []string{assetID})
	if err != nil {
		return err
	}

	// Fetch active rules to map ruleID -> rating_scope.
	rules, err := p.Engine.ListActiveForCustomer(ctx, customerID)
	if err != nil {
		return err
	}
	ruleScopes := make(map[string]string)
	for _, r := range rules {
		ruleScopes[r.RuleID] = r.RatingScope
	}

	desiredTags := make(map[string]struct{})
	for _, v := range violations {
		scope, ok := ruleScopes[v.RuleID]
		if !ok {
			scope = "unknown"
		}
		key := deliveryReadyTagPrefix + scope
		desiredTags[key] = struct{}{}
	}

	// Load current tags to compute diff.
	currentTags := make(map[string]struct{})
	for _, t := range tags {
		if strings.HasPrefix(t.TagKey, deliveryReadyTagPrefix) {
			currentTags[t.TagKey] = struct{}{}
		}
	}

	// Determine additions and deletions.
	var toAdd []string
	for key := range desiredTags {
		if _, ok := currentTags[key]; !ok {
			toAdd = append(toAdd, key)
		}
	}
	var toDel []string
	for key := range currentTags {
		if _, ok := desiredTags[key]; !ok {
			toDel = append(toDel, key)
		}
	}

	// Apply changes.
	for _, key := range toAdd {
		err := p.Tags.Upsert(ctx, repository.AssetTagUpsertInput{
			AssetID:    assetID,
			TagKey:     key,
			TagValue:   "true", // delivery_ready tags are boolean-ish
			TagType:    "system",
			SourceType: "delivery_eligibility_projector",
			SourceName: "DeliveryEligibilityProjector",
		})
		if err != nil {
			// Log but continue with other tags.
			slog.Warn("delivery eligibility projector: upsert failed", "asset_id", assetID, "tag_key", key, "err", err)
		} else {
			slog.Debug("delivery eligibility projector: added tag", "asset_id", assetID, "tag_key", key)
		}
	}
	for _, key := range toDel {
		err := p.Tags.Delete(ctx, assetID, key, "delivery_eligibility_projector")
		if err != nil {
			slog.Warn("delivery eligibility projector: delete failed", "asset_id", assetID, "tag_key", key, "err", err)
		} else {
			slog.Debug("delivery eligibility projector: removed tag", "asset_id", assetID, "tag_key", key)
		}
	}

	return nil
}

// Close releases projector resources.
func (p *DeliveryEligibilityProjector) Close() error {
	if p == nil || p.Subscriber == nil {
		return nil
	}
	return p.Subscriber.Close()
}