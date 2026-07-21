package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, path string, maintains []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	list := "[]"
	if len(maintains) > 0 {
		list = "[" + maintains[0]
		for _, m := range maintains[1:] {
			list += ", " + m
		}
		list += "]"
	}
	content := "schema_version: 1\nname: x\nstage: survey\nconsumes: []\nmaintains: " + list + "\nemits: []\nrequires_slots: []\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestCheckOwnershipAllTypesUnmaintained(t *testing.T) {
	// No manifests at all: every entry type is a gap, INCLUDING
	// standup-ledger-slot — P6 leaves suppressing that as an undecided
	// question for M4.7, not this check's to resolve silently.
	types := []string{"architecture", "conventions", "standup-ledger-slot"}
	findings, err := CheckOwnership(nil, types)
	if err != nil {
		t.Fatalf("CheckOwnership: %v", err)
	}
	if len(findings) != len(types) {
		t.Fatalf("no manifests, %d entry types = %d findings, want %d: %+v", len(types), len(findings), len(types), findings)
	}
	for _, f := range findings {
		if f.Check != "ownership-matrix" {
			t.Fatalf("finding.Check = %q, want \"ownership-matrix\"", f.Check)
		}
	}
}

func TestCheckOwnershipMaintainedTypeNotFlagged(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "skills", "atlas-foo", "manifest.yaml")
	writeManifest(t, manifestPath, []string{"architecture"})

	types := []string{"architecture", "conventions"}
	findings, err := CheckOwnership([]string{manifestPath}, types)
	if err != nil {
		t.Fatalf("CheckOwnership: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("one maintained, one not = %d findings, want exactly 1: %+v", len(findings), findings)
	}
	if findings[0].Subject == "" || findings[0].Subject == `Map entry type "architecture"` {
		t.Fatalf("the wrong (or no) entry type was flagged: %+v", findings[0])
	}
}

func TestCheckOwnershipAcrossMultipleManifests(t *testing.T) {
	// The matrix is a UNION across every installed manifest — a type
	// maintained by any one skill must not be flagged, even if most others
	// maintain nothing.
	dir := t.TempDir()
	p1 := filepath.Join(dir, "skills", "atlas-a", "manifest.yaml")
	p2 := filepath.Join(dir, "skills", "atlas-b", "manifest.yaml")
	writeManifest(t, p1, nil)
	writeManifest(t, p2, []string{"conventions"})

	findings, err := CheckOwnership([]string{p1, p2}, []string{"architecture", "conventions"})
	if err != nil {
		t.Fatalf("CheckOwnership: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("union across manifests = %d findings, want exactly 1 (only architecture): %+v", len(findings), findings)
	}
}
