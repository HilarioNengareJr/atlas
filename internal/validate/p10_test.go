package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMapEntry(t *testing.T, dir, typeName, filename, id string) {
	t.Helper()
	typeDir := filepath.Join(dir, typeName)
	if err := os.MkdirAll(typeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "schema_version: 1\nid: " + id + "\ntype: " + typeName +
		"\nsummary: test\ncreated_by_task: t1\nlast_confirmed: \"2026-07-21T00:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(typeDir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write map entry: %v", err)
	}
}

func TestCheckP10FilenameMatchesID(t *testing.T) {
	dir := t.TempDir()
	writeMapEntry(t, dir, "architecture", "loop.yaml", "loop")
	findings, err := CheckP10(dir)
	if err != nil {
		t.Fatalf("CheckP10: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("matching id/filename = %d findings, want 0: %+v", len(findings), findings)
	}
}

func TestCheckP10RenamedFileDisagreesWithID(t *testing.T) {
	dir := t.TempDir()
	writeMapEntry(t, dir, "architecture", "zoop.yaml", "loop")
	findings, err := CheckP10(dir)
	if err != nil {
		t.Fatalf("CheckP10: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("renamed file = %d findings, want exactly 1: %+v", len(findings), findings)
	}
	if findings[0].Check != "p10" {
		t.Fatalf("finding.Check = %q, want \"p10\"", findings[0].Check)
	}
}

// Break-session regression: a Map entry nested one directory level deeper
// than map/<type>/<id>.yaml expects used to be completely invisible — no
// finding, no error. Fixed 2026-07-21 — it must now be flagged.
func TestCheckP10NestedSubdirectoryIsFlagged(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "architecture", "nested_by_mistake")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "schema_version: 1\nid: loop\ntype: architecture\nsummary: t\ncreated_by_task: t1\nlast_confirmed: \"2026-07-21T00:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(nested, "loop.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	findings, err := CheckP10(dir)
	if err != nil {
		t.Fatalf("CheckP10: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("nested subdirectory = %d findings, want exactly 1: %+v", len(findings), findings)
	}
	if findings[0].Check != "p10" {
		t.Fatalf("finding.Check = %q, want \"p10\"", findings[0].Check)
	}
}

// Break-session regression: a symlinked type directory used to be
// invisible because DirEntry.IsDir() reports false for a symlink. Fixed
// 2026-07-21 by resolving via os.Stat, which follows symlinks — a
// symlinked type directory must now be walked exactly like a real one.
func TestCheckP10FollowsSymlinkedTypeDirectory(t *testing.T) {
	dir := t.TempDir()
	// The real target lives OUTSIDE dir (a separate t.TempDir()) — it must
	// only be reachable via the symlink, otherwise CheckP10's os.ReadDir(dir)
	// would also see it directly as its own sibling entry, double-counting
	// the same file.
	real := filepath.Join(t.TempDir(), "real_architecture")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// id disagrees with filename, so this must produce exactly one finding
	// once the symlinked directory is actually walked.
	content := "schema_version: 1\nid: loop\ntype: architecture\nsummary: t\ncreated_by_task: t1\nlast_confirmed: \"2026-07-21T00:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(real, "zoop.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "architecture")); err != nil {
		t.Skipf("symlinks not supported on this filesystem: %v", err)
	}

	findings, err := CheckP10(dir)
	if err != nil {
		t.Fatalf("CheckP10: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("symlinked type dir = %d findings, want exactly 1 (the id/filename mismatch inside it): %+v", len(findings), findings)
	}
}

func TestCheckP10MissingMapDirIsNotAnError(t *testing.T) {
	findings, err := CheckP10(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("CheckP10 on a missing map/ dir: %v", err)
	}
	if findings != nil {
		t.Fatalf("findings = %+v, want nil", findings)
	}
}
