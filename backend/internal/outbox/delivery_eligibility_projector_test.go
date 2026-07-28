package outbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// ---------------- shared stubs ----------------

type stubAssetRepo struct {
	assets map[string]*models.Asset
}

func (s *stubAssetRepo) Get(_ context.Context, assetID string) (*models.Asset, error) {
	return s.assets[assetID], nil
}
func (s *stubAssetRepo) GetAll(_ context.Context, assetID string) (*models.Asset, error) {
	return s.assets[assetID], nil
}
func (s *stubAssetRepo) FindExistingIDs(_ context.Context, assetIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	for _, assetID := range assetIDs {
		if s.assets[assetID] != nil {
			out[assetID] = struct{}{}
		}
	}
	return out, nil
}
func (s *stubAssetRepo) InsertNew(_ context.Context, a *models.Asset) error { return nil }
func (s *stubAssetRepo) Set(_ context.Context, a *models.Asset) error       { return nil }
func (s *stubAssetRepo) SoftDelete(_ context.Context, assetID string) error { return nil }
func (s *stubAssetRepo) ListByMcapFile(_ context.Context, mcapFileID string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) ListByLogicalAssetID(_ context.Context, logicalAssetID string) ([]*models.Asset, error) {
	return nil, nil
}
func (s *stubAssetRepo) WriteSegmentIndex(_ context.Context, a *models.Asset) error { return nil }
func (s *stubAssetRepo) ListWithFilters(_ context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	return nil, 0, nil
}
func (s *stubAssetRepo) ListDescendants(_ context.Context, assetID string) ([]*models.Asset, error) {
	return nil, nil
}

type stubTagRepo struct {
	tagsByAsset map[string][]*models.AssetTag
	upsertCalls []repository.AssetTagUpsertInput
	deleteCalls []struct {
		AssetID, TagKey, SourceType string
	}
}

func (s *stubTagRepo) Upsert(_ context.Context, in repository.AssetTagUpsertInput) error {
	s.upsertCalls = append(s.upsertCalls, in)
	return nil
}
func (s *stubTagRepo) ListByAsset(_ context.Context, assetID string) ([]*models.AssetTag, error) {
	return s.tagsByAsset[assetID], nil
}
func (s *stubTagRepo) Delete(_ context.Context, assetID, tagKey, sourceType string) error {
	s.deleteCalls = append(s.deleteCalls, struct {
		AssetID, TagKey, SourceType string
	}{assetID, tagKey, sourceType})
	return nil
}

type stubCustomerRepo struct {
	customers map[string]*models.Customer
}

func (s *stubCustomerRepo) Insert(_ context.Context, c *models.Customer) error { return nil }
func (s *stubCustomerRepo) Get(_ context.Context, customerID string) (*models.Customer, error) {
	return s.customers[customerID], nil
}
func (s *stubCustomerRepo) Update(_ context.Context, c *models.Customer) error { return nil }
func (s *stubCustomerRepo) Exists(_ context.Context, customerID string) (bool, error) {
	return false, nil
}
func (s *stubCustomerRepo) List(_ context.Context, status, slaTier, region string, limit int, cursor string) ([]*models.Customer, error) {
	return nil, nil
}

type stubRuleRepo struct {
	rules []*models.DeliveryRule
}

func (s *stubRuleRepo) Insert(_ context.Context, r *models.DeliveryRule) error { return nil }
func (s *stubRuleRepo) Get(_ context.Context, ruleID string) (*models.DeliveryRule, error) {
	return nil, nil
}
func (s *stubRuleRepo) Update(_ context.Context, r *models.DeliveryRule) error { return nil }
func (s *stubRuleRepo) Delete(_ context.Context, ruleID string) error          { return nil }
func (s *stubRuleRepo) ListActiveForCustomer(_ context.Context, customerID string) ([]*models.DeliveryRule, error) {
	return s.rules, nil
}
func (s *stubRuleRepo) List(_ context.Context, customerID string) ([]*models.DeliveryRule, error) {
	return s.rules, nil
}

type stubSubscriber struct{}

func (s *stubSubscriber) Receive(_ context.Context, _ func(context.Context, []byte) error) error {
	return nil
}
func (s *stubSubscriber) Close() error { return nil }

// ---------------- helpers ----------------

func dslMarshal(q deliveryrules.QueryDSL) json.RawMessage {
	b, _ := json.Marshal(q)
	return b
}

func tagEvent(assetID, eventType, tagKey string) []byte {
	payload, _ := json.Marshal(map[string]string{"tag_key": tagKey})
	ev := models.AssetEvent{
		AssetID:      assetID,
		EventType:    eventType,
		EventPayload: payload,
	}
	b, _ := json.Marshal(ev)
	return b
}

// ---------------- tests ----------------

func TestProjector_skipNonTagEvents(t *testing.T) {
	tagRepo := &stubTagRepo{}
	assetRepo := &stubAssetRepo{assets: map[string]*models.Asset{}}
	custRepo := &stubCustomerRepo{}
	ruleRepo := &stubRuleRepo{}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	ev := models.AssetEvent{AssetID: "a1", EventType: "algo_started"}
	data, _ := json.Marshal(ev)
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Error("expected no upserts for non-tag event")
	}
	if len(tagRepo.deleteCalls) > 0 {
		t.Error("expected no deletes for non-tag event")
	}
}

func TestProjector_skipSelfGeneratedTags(t *testing.T) {
	tagRepo := &stubTagRepo{}
	assetRepo := &stubAssetRepo{assets: map[string]*models.Asset{}}
	custRepo := &stubCustomerRepo{}
	ruleRepo := &stubRuleRepo{}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	data := tagEvent("a1", "tag.upserted", "delivery_ready:pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Error("expected no upserts for self-generated delivery_ready tag event")
	}
}

func TestProjector_emptyAssetID(t *testing.T) {
	tagRepo := &stubTagRepo{}
	assetRepo := &stubAssetRepo{assets: map[string]*models.Asset{}}
	custRepo := &stubCustomerRepo{}
	ruleRepo := &stubRuleRepo{}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	ev := models.AssetEvent{AssetID: "", EventType: "tag.upserted"}
	data, _ := json.Marshal(ev)
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Error("expected no upserts for empty asset_id")
	}
}

func TestProjector_assetNotFound(t *testing.T) {
	tagRepo := &stubTagRepo{}
	assetRepo := &stubAssetRepo{assets: map[string]*models.Asset{}}
	custRepo := &stubCustomerRepo{}
	ruleRepo := &stubRuleRepo{}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	data := tagEvent("nonexistent", "tag.upserted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Error("expected no upserts when asset not found")
	}
}

func TestProjector_ruleMatch_addsTag(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{"a1": {}},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{
		customers: map[string]*models.Customer{
			"cust1": {CustomerID: "cust1"},
		},
	}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r1", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	tagRepo.tagsByAsset["a1"] = []*models.AssetTag{
		{TagKey: "compliance.pii", TagValue: "true"},
	}

	data := tagEvent("a1", "tag.upserted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(tagRepo.upsertCalls))
	}
	call := tagRepo.upsertCalls[0]
	if call.TagKey != "delivery_ready:pii" {
		t.Errorf("expected delivery_ready:pii, got %s", call.TagKey)
	}
	if call.TagValue != "true" {
		t.Errorf("expected tag value 'true', got %s", call.TagValue)
	}
	if call.SourceType != "delivery_eligibility_projector" {
		t.Errorf("expected source_type delivery_eligibility_projector, got %s", call.SourceType)
	}
}

func TestProjector_ruleNoLongerMatches_removesTag(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{
			"a1": {
				{TagKey: "delivery_ready:pii", TagValue: "true", SourceType: "delivery_eligibility_projector"},
			},
		},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{
		customers: map[string]*models.Customer{
			"cust1": {CustomerID: "cust1"},
		},
	}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r1", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	data := tagEvent("a1", "tag.deleted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.deleteCalls) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(tagRepo.deleteCalls))
	}
	dc := tagRepo.deleteCalls[0]
	if dc.TagKey != "delivery_ready:pii" {
		t.Errorf("expected delete delivery_ready:pii, got %s", dc.TagKey)
	}
	if dc.SourceType != "delivery_eligibility_projector" {
		t.Errorf("expected source_type delivery_eligibility_projector, got %s", dc.SourceType)
	}
}

func TestProjector_idempotent_noChange(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{
			"a1": {
				{TagKey: "compliance.pii", TagValue: "true"},
				{TagKey: "delivery_ready:pii", TagValue: "true", SourceType: "delivery_eligibility_projector"},
			},
		},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{
		customers: map[string]*models.Customer{
			"cust1": {CustomerID: "cust1"},
		},
	}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r1", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	data := tagEvent("a1", "tag.upserted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Errorf("expected no upserts (tag already exists), got %d", len(tagRepo.upsertCalls))
	}
	if len(tagRepo.deleteCalls) > 0 {
		t.Errorf("expected no deletes (tag still needed), got %d", len(tagRepo.deleteCalls))
	}
}

func TestProjector_customerLookupFails_gracefulDegradation(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{"a1": {}},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{customers: map[string]*models.Customer{}}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r1", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	tagRepo.tagsByAsset["a1"] = []*models.AssetTag{
		{TagKey: "compliance.pii", TagValue: "true"},
	}

	data := tagEvent("a1", "tag.upserted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) != 1 {
		t.Fatalf("expected 1 upsert even when customer not found, got %d", len(tagRepo.upsertCalls))
	}
	if tagRepo.upsertCalls[0].TagKey != "delivery_ready:pii" {
		t.Errorf("expected delivery_ready:pii, got %s", tagRepo.upsertCalls[0].TagKey)
	}
}

func TestProjector_multipleRules_multipleTags(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{"a1": {}},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{
		customers: map[string]*models.Customer{
			"cust1": {CustomerID: "cust1"},
		},
	}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r1", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
			{RuleID: "r2", RatingScope: "sensitive", EnforceMode: "warn", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.sensitive", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	tagRepo.tagsByAsset["a1"] = []*models.AssetTag{
		{TagKey: "compliance.pii", TagValue: "true"},
		{TagKey: "sensitive", TagValue: "true"},
	}

	data := tagEvent("a1", "tag.upserted", "compliance.pii")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) != 2 {
		t.Fatalf("expected 2 upserts (one per matching rule), got %d: %+v", len(tagRepo.upsertCalls), tagRepo.upsertCalls)
	}
	keys := map[string]bool{}
	for _, c := range tagRepo.upsertCalls {
		keys[c.TagKey] = true
	}
	if !keys["delivery_ready:pii"] || !keys["delivery_ready:sensitive"] {
		t.Errorf("expected delivery_ready:pii and delivery_ready:sensitive, got %v", keys)
	}
}

func TestProjector_violationWithoutMatchingRule_usesUnknownScope(t *testing.T) {
	tagRepo := &stubTagRepo{
		tagsByAsset: map[string][]*models.AssetTag{"a1": {}},
	}
	assetRepo := &stubAssetRepo{
		assets: map[string]*models.Asset{
			"a1": {AssetID: "a1", AssetType: "clip", LifecycleState: "ready", ProjectID: "cust1"},
		},
	}
	custRepo := &stubCustomerRepo{
		customers: map[string]*models.Customer{
			"cust1": {CustomerID: "cust1"},
		},
	}
	ruleRepo := &stubRuleRepo{
		rules: []*models.DeliveryRule{
			{RuleID: "r2", RatingScope: "pii", EnforceMode: "block", QueryDSL: dslMarshal(deliveryrules.QueryDSL{
				Where: []deliveryrules.Predicate{{Field: "tag.compliance.pii", Op: "eq", Value: "true"}},
			})},
		},
	}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	tagRepo.tagsByAsset["a1"] = []*models.AssetTag{}

	data := tagEvent("a1", "tag.deleted", "some.other.tag")
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Errorf("expected no upserts when no rules match, got %d", len(tagRepo.upsertCalls))
	}
}

func TestProjector_runNilCheck(t *testing.T) {
	p := &DeliveryEligibilityProjector{}
	if err := p.Run(context.Background()); err == nil {
		t.Fatal("expected error for incomplete wiring")
	}
	if err := p.Close(); err != nil {
		t.Fatal("expected nil on close for nil subscriber")
	}
}

func TestProjector_payloadWithNoTagKey(t *testing.T) {
	tagRepo := &stubTagRepo{}
	assetRepo := &stubAssetRepo{assets: map[string]*models.Asset{}}
	custRepo := &stubCustomerRepo{}
	ruleRepo := &stubRuleRepo{}
	engine := deliveryrules.NewEngine(ruleRepo, assetRepo, tagRepo, custRepo)
	p := NewDeliveryEligibilityProjector(&stubSubscriber{}, engine, tagRepo, assetRepo, custRepo)

	payload := json.RawMessage(`{"something":"else"}`)
	ev := models.AssetEvent{
		AssetID:      "a1",
		EventType:    "tag.upserted",
		EventPayload: payload,
	}
	data, _ := json.Marshal(ev)
	err := p.handleEvent(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tagRepo.upsertCalls) > 0 {
		t.Error("expected no upserts for non-existent asset")
	}
}

func (s *stubAssetRepo) LookupDurations(context.Context, []string, int64, int64) ([]repository.DurationRow, error) {
	return nil, nil
}

func (s *stubAssetRepo) LookupCosts(context.Context, []string, time.Time, time.Time, bool) ([]repository.AssetCostRow, error) {
	return nil, nil
}
