package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copySpecDir copies the repo's real spec/*.schema.yaml into a fresh temp
// dir so a test can mutate one copy without touching the real files.
func copySpecDir(t *testing.T) string {
	t.Helper()
	src := realSpecDir(t)
	dst := t.TempDir()
	for _, name := range []string{"manifest.schema.yaml", "map.schema.yaml", "config.schema.yaml", "plan.schema.yaml", "records.schema.yaml"} {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dst
}

func TestCheckP8CleanOnRealSchemas(t *testing.T) {
	findings, err := CheckP8(realSpecDir(t))
	if err != nil {
		t.Fatalf("CheckP8: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("CheckP8 on the real, in-sync schemas = %d findings, want 0: %+v", len(findings), findings)
	}
}

func TestCheckP8CatchesDivergedEnumCopy(t *testing.T) {
	dir := copySpecDir(t)
	recordsPath := filepath.Join(dir, "records.schema.yaml")
	b, err := os.ReadFile(recordsPath)
	if err != nil {
		t.Fatalf("read records.schema.yaml copy: %v", err)
	}
	diverged := strings.Replace(string(b), "library-manifest", "library-manifest-DIVERGED", 1)
	if diverged == string(b) {
		t.Fatal("test setup broken: replacement did not change the file")
	}
	if err := os.WriteFile(recordsPath, []byte(diverged), 0o644); err != nil {
		t.Fatalf("write diverged copy: %v", err)
	}

	findings, err := CheckP8(dir)
	if err != nil {
		t.Fatalf("CheckP8: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("CheckP8 with a diverged copy = %d findings, want exactly 1: %+v", len(findings), findings)
	}
	if findings[0].Check != "p8" {
		t.Fatalf("finding.Check = %q, want \"p8\"", findings[0].Check)
	}
}
