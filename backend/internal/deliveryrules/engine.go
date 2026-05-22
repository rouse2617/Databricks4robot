package deliveryrules

import (
	"context"
	"fmt"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// Violation describes one asset blocked by one rule.
type Violation struct {
	AssetID     string `json:"asset_id"`
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	EnforceMode string `json:"enforce_mode"`
	Reason      string `json:"reason"`
}

// Engine evaluates delivery gates in PostgreSQL-backed strong consistency mode.
type Engine struct {
	rules     repository.DeliveryRuleRepository
	assets    repository.AssetRepository
	tags      repository.AssetTagRepository
	customers repository.CustomerRepository
}

func NewEngine(
	rules repository.DeliveryRuleRepository,
	assets repository.AssetRepository,
	tags repository.AssetTagRepository,
	customers repository.CustomerRepository,
) *Engine {
	return &Engine{rules: rules, assets: assets, tags: tags, customers: customers}
}

func (e *Engine) Check(ctx context.Context, customerID string, assetIDs []string) ([]Violation, error) {
	return e.checkRules(ctx, customerID, assetIDs, "block")
}

// CheckAll evaluates ALL active rules (block, warn, tag_only) and returns
// violations for every rule whose predicate matches. Used by the C2 commit
// handler to distinguish hard blocks from soft warnings.
func (e *Engine) CheckAll(ctx context.Context, customerID string, assetIDs []string) ([]Violation, error) {
	return e.checkRules(ctx, customerID, assetIDs, "")
}

// checkRules is the shared implementation. When filterMode is non-empty only
// rules with that enforce_mode are evaluated; when empty all active rules are
// evaluated.
func (e *Engine) checkRules(ctx context.Context, customerID string, assetIDs []string, filterMode string) ([]Violation, error) {
	if e.rules == nil {
		return nil, nil
	}
	rules, err := e.rules.ListActiveForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	var customer *models.Customer
	if e.customers != nil {
		customer, err = e.customers.Get(ctx, customerID)
		if err != nil {
			return nil, err
		}
	}

	type compiledRule struct {
		id, name, mode string
		dsl            *QueryDSL
	}
	var compiled []compiledRule
	for _, r := range rules {
		if filterMode != "" && r.EnforceMode != filterMode {
			continue
		}
		dsl, err := ParseQueryDSL(r.QueryDSL)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.RuleID, err)
		}
		compiled = append(compiled, compiledRule{
			id: r.RuleID, name: r.Name, mode: r.EnforceMode, dsl: dsl,
		})
	}

	var violations []Violation
	for _, assetID := range assetIDs {
		snap, err := e.loadSnapshot(ctx, assetID)
		if err != nil {
			return nil, err
		}
		for _, cr := range compiled {
			hit, err := Matches(snap, cr.dsl)
			if err != nil {
				return nil, err
			}
			if hit {
				violations = append(violations, Violation{
					AssetID: assetID, RuleID: cr.id, RuleName: cr.name,
					EnforceMode: cr.mode,
					Reason:      fmt.Sprintf("asset matches rule %q", cr.name),
				})
			}
		}
		if customer != nil {
			violations = append(violations, MatchExcludeTagsForAsset(assetID, customer, snap)...)
		}
	}
	return violations, nil
}

func (e *Engine) loadSnapshot(ctx context.Context, assetID string) (AssetSnapshot, error) {
	if e.assets == nil {
		return AssetSnapshot{AssetID: assetID}, nil
	}
	a, err := e.assets.Get(ctx, assetID)
	if err != nil {
		return AssetSnapshot{}, err
	}
	if a == nil {
		return AssetSnapshot{}, fmt.Errorf("asset %s not found", assetID)
	}
	var tags []*models.AssetTag
	if e.tags != nil {
		tags, err = e.tags.ListByAsset(ctx, assetID)
		if err != nil {
			return AssetSnapshot{}, err
		}
	}
	return BuildSnapshot(a, tags), nil
}

// MatchExcludeTagsForAsset checks customer.exclude_tags against loaded tags.
func MatchExcludeTagsForAsset(assetID string, c *models.Customer, snap AssetSnapshot) []Violation {
	var out []Violation
	for _, raw := range c.ExcludeTags {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := m["key"].(string)
		val, _ := m["value"].(string)
		if key == "" {
			continue
		}
		dsl := &QueryDSL{
			Where: []Predicate{{Field: "tag." + key, Op: "eq", Value: val}},
		}
		hit, err := Matches(snap, dsl)
		if err != nil || !hit {
			continue
		}
		out = append(out, Violation{
			AssetID: assetID, RuleID: "customer.exclude_tags", RuleName: "customer exclude_tags",
			EnforceMode: "block",
			Reason:      fmt.Sprintf("tag %s=%s excluded for customer", key, val),
		})
	}
	return out
}
