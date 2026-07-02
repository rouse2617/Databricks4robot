package backfill

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func strPtr(v string) *string { return &v }

func TestBackfillItemLedgerStatus(t *testing.T) {
	status, msg := backfillItemLedgerStatus(models.BackfillItem{
		Status:       "failed",
		ErrorMessage: strPtr("boom"),
	})
	if status != "Failed" || msg != "boom" {
		t.Fatalf("got status=%q msg=%q", status, msg)
	}
}

func TestBackfillItemLedgerStatus_Pending(t *testing.T) {
	status, msg := backfillItemLedgerStatus(models.BackfillItem{Status: "pending"})
	if status != "Pending" || msg != "" {
		t.Fatalf("got status=%q msg=%q", status, msg)
	}
}

func TestBackfillItemLedgerStatus_Running(t *testing.T) {
	status, _ := backfillItemLedgerStatus(models.BackfillItem{Status: "running"})
	if status != "Running" {
		t.Fatalf("got status=%q", status)
	}
}
