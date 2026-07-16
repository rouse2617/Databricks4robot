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

func TestDeriveWithMessage_TerminalErrorProjectsRunningNode(t *testing.T) {
	rows := []models.PipelineRunAssetNode{
		{PipelineNodeID: "step-a", DisplayName: "Read", Status: "Succeeded"},
		{PipelineNodeID: "step-b", DisplayName: "Transform", Status: "Running"},
		{PipelineNodeID: "step-c", DisplayName: "Write", Status: "Pending"},
	}
	got := DeriveWithMessage(rows, "Error", "Argo workflow was cleaned up")
	if got.FocusNodeName != "Transform" || got.FocusStatus != "Error" {
		t.Fatalf("unexpected focus: %+v", got)
	}
	if got.Message != "Argo workflow was cleaned up" {
		t.Fatalf("message = %q", got.Message)
	}
}

func TestDerive_EmptyRowsPending(t *testing.T) {
	got := Derive(nil, "Pending")
	if got == nil || got.Label != "未开始" {
		t.Fatalf("unexpected: %+v", got)
	}
}

// CYB-3491: a terminally-failed run must NEVER present as 进行中, regardless
// of what the (stale) node snapshot says. The unschedulable class leaves node
// rows stuck at "Pending" — pods never started.
func TestDeriveWithMessage_TerminalFailureOverridesStaleNodes(t *testing.T) {
	msg := "Kubernetes 调度失败:step-nw-delivery Unschedulable"

	// Node row stuck Pending (pod never scheduled) + run Error → Error focus
	// carrying the run message, not 进行中.
	got := DeriveWithMessage([]models.PipelineRunAssetNode{
		{PipelineNodeID: "step-nw-delivery", DisplayName: "step-nw-delivery", Status: "Pending"},
	}, "Error", msg)
	if got == nil || got.FocusStatus != "Error" {
		t.Fatalf("pending-node terminal run = %+v, want Error focus", got)
	}
	if got.FocusNodeName != "step-nw-delivery" || got.Message != msg {
		t.Fatalf("focus/message = %+v, want node blamed with run message", got)
	}

	// Every node row already terminal-succeeded but the run failed (DAG-level
	// failure): generic 异常 label with the run message — still never 进行中.
	got = DeriveWithMessage([]models.PipelineRunAssetNode{
		{PipelineNodeID: "step-1", Status: "Succeeded"},
	}, "Failed", "dag template failed")
	if got == nil || got.FocusStatus != "Error" || got.Label != "异常" {
		t.Fatalf("dag-level failure = %+v, want 异常 label", got)
	}

	// Regression guard: a genuinely ACTIVE run with pending nodes keeps the
	// in-progress presentation.
	got = DeriveWithMessage([]models.PipelineRunAssetNode{
		{PipelineNodeID: "step-1", Status: "Pending"},
	}, "Running", "")
	if got == nil || got.Label != "进行中" {
		t.Fatalf("active run = %+v, want 进行中", got)
	}
}
