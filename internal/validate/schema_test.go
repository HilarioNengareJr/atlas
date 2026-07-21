package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realSpecDir resolves this repo's real spec/ directory from the test's
// working directory (internal/validate) — the schemas under test are the
// project's actual, live schemas, not copies.
func realSpecDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "spec")
	if _, err := os.Stat(filepath.Join(dir, "manifest.schema.yaml")); err != nil {
		t.Fatalf("could not find the repo's real spec/ dir at %s: %v", dir, err)
	}
	return dir
}

func TestLoadSchemasCompilesAllFive(t *testing.T) {
	set, err := LoadSchemas(realSpecDir(t))
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	for _, k := range []Kind{KindManifest, KindMap, KindConfig, KindPlan, KindRecords} {
		if _, ok := set.schemas[k]; !ok {
			t.Errorf("schema for kind %q did not compile", k)
		}
	}
}

func TestDispatchKindByDirectoryConvention(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Kind
	}{
		{"manifest under skills/", "skills/atlas-plan/manifest.yaml", KindManifest},
		{"map entry", "map/architecture/loop.yaml", KindMap},
		{"config", "atlas.config", KindConfig},
		{"config nested", "some/dir/atlas.config", KindConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DispatchKind(tt.path, "")
			if err != nil {
				t.Fatalf("DispatchKind(%q, \"\"): %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("DispatchKind(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDispatchKindWithoutConventionRequiresOverride(t *testing.T) {
	if _, err := DispatchKind("some/random/plan.yaml", ""); err == nil {
		t.Fatal("expected an error for a plan/records-shaped path with no --schema override")
	}
	got, err := DispatchKind("some/random/plan.yaml", "plan")
	if err != nil || got != KindPlan {
		t.Fatalf("DispatchKind with --schema=plan = (%q, %v), want (KindPlan, nil)", got, err)
	}
}

func TestDispatchKindExplicitOverrideAlwaysWins(t *testing.T) {
	// Even a path that WOULD match the manifest convention should honour an
	// explicit override for a different kind — the M1 example fixtures
	// (spec/examples/manifests/*.yaml) rely on exactly this to be checkable
	// at all, since they don't live at skills/<name>/manifest.yaml.
	got, err := DispatchKind("spec/examples/manifests/architect.yaml", "manifest")
	if err != nil || got != KindManifest {
		t.Fatalf("explicit override = (%q, %v), want (KindManifest, nil)", got, err)
	}
}

func TestDispatchKindRejectsUnknownOverride(t *testing.T) {
	if _, err := DispatchKind("x.yaml", "nonsense"); err == nil {
		t.Fatal("expected an error for an unrecognised --schema value")
	}
}

func TestCheckInstanceRealM1Examples(t *testing.T) {
	set, err := LoadSchemas(realSpecDir(t))
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	examples := []string{"architect.yaml", "build.yaml", "review.yaml"}
	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "spec", "examples", "manifests", name)
			findings, err := set.CheckInstance(path, "manifest")
			if err != nil {
				t.Fatalf("CheckInstance(%s): %v", path, err)
			}
			if len(findings) != 0 {
				t.Fatalf("CheckInstance(%s) = %d findings, want 0 (matches the M4 build-plan's exit criterion): %+v", path, len(findings), findings)
			}
		})
	}
}

// Review fix: schema-shape findings must carry a real source line, not just
// a JSON-pointer field path (the plan's stated format is file:line).
func TestCheckInstanceReportsRealLineNumbers(t *testing.T) {
	set, err := LoadSchemas(realSpecDir(t))
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	// schema_version wrong on line 1; consumes wrong-typed on line 5.
	content := "schema_version: \"1\"\nname: x\nstage: chart\nconsumes: \"not-an-array\"\nmaintains: []\nemits: []\nrequires_slots: []\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	findings, err := set.CheckInstance(path, "manifest")
	if err != nil {
		t.Fatalf("CheckInstance: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings for a corrupted instance")
	}
	sawRealLine := false
	for _, f := range findings {
		if f.Line > 0 {
			sawRealLine = true
		}
	}
	if !sawRealLine {
		t.Fatalf("no finding carried a resolved Line — expected at least one: %+v", findings)
	}
}

// Review fix: CheckInstanceAsKind must not re-dispatch — it validates
// against exactly the Kind given, regardless of the file's own path shape.
// This is the function directory-mode callers use precisely so they never
// hit the mis-dispatch bug CheckInstance(path, overrideThatApppliesToEveryFile)
// caused.
func TestCheckInstanceAsKindDoesNotDispatch(t *testing.T) {
	set, err := LoadSchemas(realSpecDir(t))
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	// A real, valid manifest, checked explicitly AS a manifest — must be
	// clean regardless of what directory it's sitting in.
	path := filepath.Join("..", "..", "spec", "examples", "manifests", "architect.yaml")
	findings, err := set.CheckInstanceAsKind(path, KindManifest)
	if err != nil {
		t.Fatalf("CheckInstanceAsKind: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("valid manifest checked as KindManifest = %d findings, want 0: %+v", len(findings), findings)
	}
}

// Break-session regression: a stray '---' document separator used to
// silently truncate to the first document, with zero indication a second
// document existed. Fixed 2026-07-21 — this must now be a clear error.
func TestDecodeYAMLFileRejectsMultipleDocuments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multidoc.yaml")
	content := "a: 1\n---\nb: 2\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := decodeYAMLFile(path)
	if err == nil {
		t.Fatal("expected an error for a multi-document YAML file, got none")
	}
	if !strings.Contains(err.Error(), "more than one YAML document") {
		t.Fatalf("error = %v, want it to mention multiple documents", err)
	}
}

func TestDecodeYAMLFileEmptyFileStillDecodesToNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	v, err := decodeYAMLFile(path)
	if err != nil {
		t.Fatalf("decodeYAMLFile on an empty file: %v", err)
	}
	if v != nil {
		t.Fatalf("decoded value = %v, want nil (matches YAML null semantics)", v)
	}
}

func TestCheckInstanceFailsCorruptedManifest(t *testing.T) {
	set, err := LoadSchemas(realSpecDir(t))
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupted.yaml")
	// schema_version is a string, not the required int 1; every other
	// required field is also missing.
	if err := os.WriteFile(path, []byte("schema_version: \"1\"\nname: x\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	findings, err := set.CheckInstance(path, "manifest")
	if err != nil {
		t.Fatalf("CheckInstance: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("corrupted instance produced 0 findings, want at least one")
	}
	for _, f := range findings {
		if f.What == "" || f.Why == "" || f.Next == "" {
			t.Fatalf("finding missing a part of the three-part standard: %+v", f)
		}
	}
}
