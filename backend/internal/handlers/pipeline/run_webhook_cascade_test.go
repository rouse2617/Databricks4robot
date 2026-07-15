package pipeline

import "testing"

// The webhook cascade (CYB-3078) only fires for terminal run phases; this guards
// the terminal-status classification used to decide whether to sync the parent
// batch.
func TestIsTerminalRunStatus(t *testing.T) {
	terminal := []string{"Succeeded", "Failed", "Error"}
	for _, s := range terminal {
		if !isTerminalRunStatus(s) {
			t.Errorf("isTerminalRunStatus(%q) = false, want true", s)
		}
	}
	nonTerminal := []string{"Running", "Pending", "", "succeeded", "unknown"}
	for _, s := range nonTerminal {
		if isTerminalRunStatus(s) {
			t.Errorf("isTerminalRunStatus(%q) = true, want false", s)
		}
	}
}
