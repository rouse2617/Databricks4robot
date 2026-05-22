package repository

import (
	"context"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// DeliveryRuleRepository persists delivery compliance rules.
type DeliveryRuleRepository interface {
	Insert(ctx context.Context, rule *models.DeliveryRule) error
	ListActiveForCustomer(ctx context.Context, customerID string) ([]*models.DeliveryRule, error)
	List(ctx context.Context, customerID string) ([]*models.DeliveryRule, error)
}
