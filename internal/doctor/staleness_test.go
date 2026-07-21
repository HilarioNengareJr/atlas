package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// referenceNow is the fixed instant every staleness test measures against.
// Fixed rather than time.Now() so the fixture dates below mean exactly one
// thing forever, and a test that passes today still passes in a year.
var referenceNow = time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)

// mapEntry writes a real Map entry file at map/<entryType>/<id>.yaml under
// root, confirmed at the given time.
func mapEntry(t *testing.T, root, entryType, id string, confirmed time.Time) {
	t.Helper()
	writeMapFile(t, root, entryType, id+".yaml", fmt.Sprintf(
		`schema_version: 1
id: %s
type: %s
summary: A real entry, shaped exactly as spec/map.schema.yaml requires.
created_by_task: t1
last_confirmed: "%s"
`, id, entryType, confirmed.Format(time.RFC3339)))
}

// writeMapFile writes arbitrary content to map/<entryType>/<name> under root.
func writeMapFile(t *testing.T, root, entryType, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "map", entryType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestCheckStalenessFreshEntryIsClean(t *testing.T) {
	root := t.TempDir()
	mapEntry(t, root, "architecture", "loop", referenceNow.Add(-24*time.Hour))
	if f := CheckStaleness(filepath.Join(root, "map"), referenceNow); len(f) != 0 {
		t.Fatalf("an entry confirmed yesterday is fresh and must produce no findings, got: %v", f)
	}
}

func TestCheckStalenessFlagsAnOldEntry(t *testing.T) {
	root := t.TempDir()
	mapEntry(t, root, "architecture", "loop", referenceNow.Add(-90*24*time.Hour))

	findings := CheckStaleness(filepath.Join(root, "map"), referenceNow)
	if len(findings) != 1 {
		t.Fatalf("want exactly 1 finding for a 90-day-old entry, got %d: %v", len(findings), findings)
	}
	if findings[0].Check != "map-staleness" {
		t.Errorf("Check = %q, want %q", findings[0].Check, "map-staleness")
	}
	if !strings.Contains(findings[0].What, "90 days ago") {
		t.Errorf("the what-line should state the real age, got: %q", findings[0].What)
	}
}

// The boundary is worth pinning: StaleAfter is the last fresh moment, not the
// first stale one. Without a test, the comparison silently flips on a refactor.
func TestCheckStalenessBoundaryIsInclusive(t *testing.T) {
	t.Run("exactly at the threshold is still fresh", func(t *testing.T) {
		root := t.TempDir()
		mapEntry(t, root, "architecture", "loop", referenceNow.Add(-StaleAfter))
		if f := CheckStaleness(filepath.Join(root, "map"), referenceNow); len(f) != 0 {
			t.Fatalf("an entry exactly %s old must still be fresh, got: %v", StaleAfter, f)
		}
	})
	t.Run("one second past the threshold is stale", func(t *testing.T) {
		root := t.TempDir()
		mapEntry(t, root, "architecture", "loop", referenceNow.Add(-StaleAfter-time.Second))
		if f := CheckStaleness(filepath.Join(root, "map"), referenceNow); len(f) != 1 {
			t.Fatalf("one second past %s must be stale, got %d findings", StaleAfter, len(f))
		}
	})
}

func TestCheckStalenessUndatableEntriesAreFindingsNotSilentSkips(t *testing.T) {
	cases := map[string]string{
		"no last_confirmed": `schema_version: 1
id: loop
type: architecture
summary: An entry with no date at all.
created_by_task: t1
`,
		"unparseable last_confirmed": `schema_version: 1
id: loop
type: architecture
summary: An entry dated in a shape nothing can parse.
created_by_task: t1
last_confirmed: "not-a-timestamp"
`,
		"not a mapping": "- just\n- a\n- list\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeMapFile(t, root, "architecture", "loop.yaml", content)
			findings := CheckStaleness(filepath.Join(root, "map"), referenceNow)
			if len(findings) != 1 {
				t.Fatalf("an entry whose age can't be determined must be reported, not skipped — got %d findings", len(findings))
			}
			if !strings.Contains(findings[0].What, "Could not determine") {
				t.Errorf("what-line should say the age couldn't be determined, got: %q", findings[0].What)
			}
		})
	}
}

func TestCheckStalenessMissingMapDirectoryIsAFinding(t *testing.T) {
	findings := CheckStaleness(filepath.Join(t.TempDir(), "map"), referenceNow)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding for a missing map/, got %d: %v", len(findings), findings)
	}
	if !strings.Contains(findings[0].What, "No map/ directory") {
		t.Errorf("what-line should name the missing directory, got: %q", findings[0].What)
	}
}

func TestCheckStalenessMapAsAFileIsAFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "map"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write map: %v", err)
	}
	findings := CheckStaleness(filepath.Join(root, "map"), referenceNow)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding when map/ is a file, got %d: %v", len(findings), findings)
	}
}

func TestCheckStalenessIgnoresNonYAMLAndStrayFiles(t *testing.T) {
	root := t.TempDir()
	mapEntry(t, root, "architecture", "loop", referenceNow)
	writeMapFile(t, root, "architecture", "README.md", "Notes for humans, not an entry.\n")
	writeMapFile(t, root, "architecture", ".DS_Store", "junk\n")
	if err := os.WriteFile(filepath.Join(root, "map", "stray.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write stray: %v", err)
	}
	if f := CheckStaleness(filepath.Join(root, "map"), referenceNow); len(f) != 0 {
		t.Fatalf("non-YAML files are not Map entries and must not be checked, got: %v", f)
	}
}

func TestCheckStalenessReportsEveryStaleEntryInStableOrder(t *testing.T) {
	root := t.TempDir()
	old := referenceNow.Add(-60 * 24 * time.Hour)
	mapEntry(t, root, "architecture", "loop", old)
	mapEntry(t, root, "conventions", "naming", old)
	mapEntry(t, root, "decisions", "install-name", old)

	first := CheckStaleness(filepath.Join(root, "map"), referenceNow)
	if len(first) != 3 {
		t.Fatalf("want 3 findings, got %d", len(first))
	}
	// Go map iteration is random; a report that reshuffles between identical
	// runs is one nobody can diff.
	for i := 0; i < 5; i++ {
		again := CheckStaleness(filepath.Join(root, "map"), referenceNow)
		for j := range first {
			if first[j].Subject != again[j].Subject {
				t.Fatalf("finding order is not stable across runs: %q vs %q", first[j].Subject, again[j].Subject)
			}
		}
	}
}
