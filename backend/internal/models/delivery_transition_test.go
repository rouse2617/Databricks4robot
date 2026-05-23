package models

import "testing"

func TestValidateTransition_Created(t *testing.T) {
	if err := ValidateTransition("", DeliveryStatusPending); err != nil {
		t.Fatalf("expected pending from creation, got %v", err)
	}
	if err := ValidateTransition("", DeliveryStatusDelivered); err != nil {
		t.Fatalf("expected delivered from creation, got %v", err)
	}
}

func TestValidateTransition_Pending(t *testing.T) {
	if err := ValidateTransition(DeliveryStatusPending, DeliveryStatusDelivered); err != nil {
		t.Fatalf("expected delivered from pending, got %v", err)
	}
	if err := ValidateTransition(DeliveryStatusPending, DeliveryStatusCancelled); err != nil {
		t.Fatalf("expected cancelled from pending, got %v", err)
	}
	if err := ValidateTransition(DeliveryStatusPending, DeliveryStatusArchived); err == nil {
		t.Fatal("expected error for pending->archived")
	}
}

func TestValidateTransition_Archived(t *testing.T) {
	// CYB-1052 / CYB-1123: delivered -> accepted -> archived is valid
	if err := ValidateTransition(DeliveryStatusDelivered, DeliveryStatusAccepted); err != nil {
		t.Fatalf("expected accepted from delivered, got %v", err)
	}
	if err := ValidateTransition(DeliveryStatusDelivered, DeliveryStatusArchived); err != nil {
		t.Fatalf("expected archived from delivered, got %v", err)
	}
	if err := ValidateTransition(DeliveryStatusAccepted, DeliveryStatusArchived); err != nil {
		t.Fatalf("expected archived from accepted, got %v", err)
	}
	// Reverse is not allowed
	if err := ValidateTransition(DeliveryStatusArchived, DeliveryStatusDelivered); err == nil {
		t.Fatal("expected error for archived->delivered")
	}
}

func TestValidateTransition_Disallowed(t *testing.T) {
	if err := ValidateTransition(DeliveryStatusDelivered, DeliveryStatusPending); err == nil {
		t.Fatal("expected error for delivered->pending")
	}
	if err := ValidateTransition(DeliveryStatusCancelled, DeliveryStatusDelivered); err == nil {
		t.Fatal("expected error for cancelled->delivered")
	}
	if err := ValidateTransition(DeliveryStatusAccepted, DeliveryStatusDelivered); err == nil {
		t.Fatal("expected error for accepted->delivered")
	}
}
