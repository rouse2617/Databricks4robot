package repository

import (
	"context"
	"errors"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

// ErrDuplicateCustomerID is returned when customer_id already exists.
var ErrDuplicateCustomerID = errors.New("duplicate customer id")

// CustomerRepository persists customer master data.
type CustomerRepository interface {
	Insert(ctx context.Context, c *models.Customer) error
	Get(ctx context.Context, customerID string) (*models.Customer, error)
	Update(ctx context.Context, c *models.Customer) error
	Exists(ctx context.Context, customerID string) (bool, error)
	List(ctx context.Context, status, slaTier, region string, limit int, cursor string) ([]*models.Customer, error)
}
