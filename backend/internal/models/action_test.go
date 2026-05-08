package models

import "testing"

func TestIsValidActionSourceType(t *testing.T) {
	for _, s := range []string{ActionSourceHuman, ActionSourceAlgo, ActionSourceRule, ActionSourceSystem} {
		if !IsValidActionSourceType(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	for _, s := range []string{"", "robot", "annotator"} {
		if IsValidActionSourceType(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}
