package validate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFindingStringHasAllThreeParts(t *testing.T) {
	f := Finding{Check: "schema", File: "x.yaml", Field: "/name", What: "w", Why: "y", Next: "n"}
	s := f.String()
	for _, want := range []string{"x.yaml", "/name", "what: w", "why:  y", "next: n"} {
		if !strings.Contains(s, want) {
			t.Fatalf("Finding.String() = %q, missing %q", s, want)
		}
	}
}

func TestFindingLocationFallsBackToSubject(t *testing.T) {
	f := Finding{Check: "ownership-matrix", Subject: "Map entry type \"architecture\""}
	if got := f.location(); got != f.Subject {
		t.Fatalf("location() = %q, want the Subject %q (no File set)", got, f.Subject)
	}
}

func TestRenderHumanCleanPass(t *testing.T) {
	out := RenderHuman(nil)
	if !strings.Contains(out, "clean") {
		t.Fatalf("RenderHuman(nil) = %q, want an explicit clean-pass message", out)
	}
}

func TestRenderJSONShapeAndCleanFlag(t *testing.T) {
	out, err := RenderJSON(nil)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var doc struct {
		Clean    bool      `json:"clean"`
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("RenderJSON output not valid JSON: %v", err)
	}
	if !doc.Clean {
		t.Fatal("RenderJSON(nil).clean = false, want true")
	}
	if doc.Findings == nil {
		t.Fatal("RenderJSON(nil).findings = null, want an empty array (not omitted/null)")
	}

	withFindings, err := RenderJSON([]Finding{{Check: "schema", File: "x", What: "w", Why: "y", Next: "n"}})
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var doc2 struct {
		Clean    bool      `json:"clean"`
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal(withFindings, &doc2); err != nil {
		t.Fatalf("RenderJSON output not valid JSON: %v", err)
	}
	if doc2.Clean {
		t.Fatal("RenderJSON with a finding reports clean=true")
	}
	if len(doc2.Findings) != 1 {
		t.Fatalf("RenderJSON findings length = %d, want 1", len(doc2.Findings))
	}
}
