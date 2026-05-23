package models

import "fmt"

// validTransitions defines allowed delivery status changes.
// Key is the "from" status (empty string = creation), value is the set of
// allowed "to" statuses.
var validTransitions = map[DeliveryStatus]map[DeliveryStatus]bool{
	"": {
		DeliveryStatusPending:   true, // draft creation
		DeliveryStatusDelivered: true, // one-step commit (existing path)
	},
	DeliveryStatusPending: {
		DeliveryStatusDelivered: true, // C2 commit
		DeliveryStatusCancelled: true, // cancel draft
	},
	DeliveryStatusDelivered: {
		DeliveryStatusArchived: true, // archive after delivery
	},
	DeliveryStatusAccepted: {
		DeliveryStatusArchived: true, // archive after acceptance
	},
}

// ValidateTransition returns nil when from->to is a legal delivery state change,
// or an error describing the violation.
func ValidateTransition(from, to DeliveryStatus) error {
	targets, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("no transitions allowed from status %q", from)
	}
	if !targets[to] {
		return fmt.Errorf("transition from %q to %q is not allowed", from, to)
	}
	return nil
}
