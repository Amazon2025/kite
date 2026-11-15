package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// A check that has been turned off says so rather than reaching GitHub.
func TestVersionCheckIsOffWhenTurnedOff(t *testing.T) {
	t.Setenv("KITE_NO_UPDATE_CHECK", "1")

	out := runKite(t, t.TempDir(), "version", "--check")
	if !strings.Contains(out, "update checks are off") {
		t.Errorf("output:\n%s", out)
	}
}

// A script reads the same answer out of --json.
func TestVersionCheckSaysItIsOffInJSON(t *testing.T) {
	t.Setenv("KITE_NO_UPDATE_CHECK", "true")

	out := runKite(t, t.TempDir(), "version", "--check", "--json")
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got["check"] != "off" {
		t.Errorf("check = %v, want off", got["check"])
	}
	if got["version"] == nil {
		t.Errorf("the answer no longer says which version it is: %v", got)
	}
}
