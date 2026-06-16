package batchprogress

import (
	"testing"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

func TestDerive_SucceededShowsLastNode(t *testing.T) {
	rows := []models.PipelineRunAssetNode{
		{PipelineNodeID: "step-a", DisplayName: "Read", Status: "Succeeded"},
		{PipelineNodeID: "step-b", DisplayName: "Transform", Status: "Succeeded"},
		{PipelineNodeID: "step-c", DisplayName: "Write", Status: "Succeeded"},
	}
	got := Derive(rows, "Succeeded")
	if got == nil {
		t.Fatal("expected progress")
	}
	if got.Label != "" {
		t.Fatalf("expected focus node, got label %q", got.Label)
	}
	if got.FocusNodeName != "Write" || got.FocusStatus != "Succeeded" {
		t.Fatalf("unexpected focus: %+v", got)
	}
}

func TestDerive_FailedTakesPriorityOverSucceeded(t *testing.T) {
	rows := []models.PipelineRunAssetNode{
		{PipelineNodeID: "step-a", DisplayName: "Read", Status: "Succeeded"},
		{PipelineNodeID: "step-b", DisplayName: "Transform", Status: "Failed", Message: "boom"},
	}
	got := Derive(rows, "Failed")
	if got.FocusNodeName != "Transform" || got.FocusStatus != "Failed" {
		t.Fatalf("unexpected focus: %+v", got)
	}
}

func TestDerive_RunningShowsCurrentNode(t *testing.T) {
	rows := []models.PipelineRunAssetNode{
		{PipelineNodeID: "step-a", DisplayName: "Read", Status: "Succeeded"},
		{PipelineNodeID: "step-b", DisplayName: "Transform", Status: "Running"},
	}
	got := Derive(rows, "Running")
	if got.FocusNodeName != "Transform" || got.FocusStatus != "Running" {
		t.Fatalf("unexpected focus: %+v", got)
	}
}

func TestDerive_EmptyRowsPending(t *testing.T) {
	got := Derive(nil, "Pending")
	if got == nil || got.Label != "未开始" {
		t.Fatalf("unexpected: %+v", got)
	}
}
