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

// --short prints the version and nothing else, for a script to read.
func TestVersionShortPrintsTheVersionAlone(t *testing.T) {
	out := runKite(t, t.TempDir(), "version", "--short")

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Fatalf("want one line, got %d:\n%s", len(lines), out)
	}
	// A test binary carries no stamped version, so it reports itself as dev.
	if lines[0] != "dev" {
		t.Errorf("short output = %q, want dev", lines[0])
	}
}
