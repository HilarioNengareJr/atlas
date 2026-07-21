package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// realSchemas compiles the actual spec/*.schema.yaml from this repo — not a
// stub, not a fixture schema. Doctor's config check is only meaningful
// against the schemas the project really ships.
func realSchemas(t *testing.T) *validate.SchemaSet {
	t.Helper()
	specDir := filepath.Join("..", "..", "spec")
	if _, err := os.Stat(filepath.Join(specDir, "config.schema.yaml")); err != nil {
		t.Fatalf("could not find the real spec/ at %s: %v", specDir, err)
	}
	schemas, err := validate.LoadSchemas(specDir)
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	return schemas
}

// writeFile writes content at path under root, creating parent directories.
func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// checksIn lists the Check field of every finding, for assertions that care
// about which checks fired rather than their exact wording.
func checksIn(findings []validate.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Check)
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// cleanRepo builds a repo in which every single doctor check passes: git
// ignores .env, the config is schema-valid, the one secret it declares is
// present, and the Map entry was confirmed yesterday.
func cleanRepo(t *testing.T) string {
	t.Helper()
	root := gitRepo(t, ".env\n")
	writeFile(t, root, "atlas.config", `schema_version: 1
mcps:
  postgres:
    slot: build
    env:
      - POSTGRES_URL
`)
	writeFile(t, root, ".env", "POSTGRES_URL=postgres://localhost/dev\n")
	mapEntry(t, root, "architecture", "loop", referenceNow.Add(-24*time.Hour))
	return root
}

// brokenRepo carries every condition the plan's exit criterion names, at
// once: a schema-invalid MCP declaration, a declared secret that isn't set,
// an unignored .env, and a 90-day-old Map entry.
func brokenRepo(t *testing.T) string {
	t.Helper()
	root := gitRepo(t, "*.log\n") // .env deliberately NOT ignored
	writeFile(t, root, "atlas.config", `schema_version: 1
mcps:
  postgres:
    slot: not-a-real-slot
    env:
      - POSTGRES_URL
`)
	mapEntry(t, root, "architecture", "loop", referenceNow.Add(-90*24*time.Hour))
	return root
}

func TestRunCleanRepoHasNoFindings(t *testing.T) {
	findings := Run(Options{
		Dir:     cleanRepo(t),
		Schemas: realSchemas(t),
		Now:     referenceNow,
	})
	if len(findings) != 0 {
		t.Fatalf("a healthy repo must produce no findings, got %d:\n%s",
			len(findings), validate.RenderHumanFor("doctor", findings))
	}
}

// The plan's stated exit criterion, checked literally.
func TestRunBrokenRepoReportsEveryCondition(t *testing.T) {
	findings := Run(Options{
		Dir:     brokenRepo(t),
		Schemas: realSchemas(t),
		Now:     referenceNow,
	})

	// "schema" rather than "config" for the invalid slot: a schema-shape
	// violation is labelled by the check that found it (validate's schema
	// check, reused here), not by the command that ran it — the same
	// finding reads identically from atlas validate and atlas doctor.
	checks := checksIn(findings)
	for _, want := range []string{"schema", "secrets", "env-gitignored", "map-staleness"} {
		if !contains(checks, want) {
			t.Errorf("missing a %q finding; got checks %v:\n%s",
				want, checks, validate.RenderHumanFor("doctor", findings))
		}
	}

	for _, f := range findings {
		if f.What == "" || f.Why == "" || f.Next == "" {
			t.Errorf("every finding must carry what/why/next (code-standards.md Error Handling), got %+v", f)
		}
	}
}

// Every check runs regardless of what the ones before it found. Stopping at
// the first problem would hide the rest, which is the opposite of a health
// check's job.
func TestRunDoesNotStopAtTheFirstProblem(t *testing.T) {
	findings := Run(Options{
		Dir:     brokenRepo(t),
		Schemas: realSchemas(t),
		Now:     referenceNow,
	})
	if len(checksIn(findings)) < 4 {
		t.Fatalf("want findings from all four checks, got %v", checksIn(findings))
	}
}

// The P14 conflict, tested: on a real project repo Atlas's spec/ isn't
// reachable, and the checks that don't need schemas must still run. This is
// the case that made doctor degrade rather than abort.
func TestRunWithoutSchemasStillChecksSecretsAndGitignoreAndMap(t *testing.T) {
	findings := Run(Options{
		Dir:                brokenRepo(t),
		Schemas:            nil,
		SchemasUnavailable: "Atlas's spec/ directory isn't reachable from here",
		Now:                referenceNow,
	})

	checks := checksIn(findings)
	for _, want := range []string{"secrets", "env-gitignored", "map-staleness"} {
		if !contains(checks, want) {
			t.Errorf("check %q must still run without schemas; got %v", want, checks)
		}
	}

	var explained bool
	for _, f := range findings {
		if strings.Contains(f.What, "Could not schema-check atlas.config") {
			explained = true
			if !strings.Contains(f.What, "isn't reachable") {
				t.Errorf("the finding should carry the reason it was given, got: %q", f.What)
			}
		}
	}
	if !explained {
		t.Error("an unavailable schema set must be reported as its own finding, not silently skipped")
	}
}

func TestRunMissingConfigIsAFindingNotACrash(t *testing.T) {
	root := gitRepo(t, ".env\n")
	mapEntry(t, root, "architecture", "loop", referenceNow)

	findings := Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow})
	if len(findings) != 1 {
		t.Fatalf("want exactly 1 finding (the missing config), got %d: %v", len(findings), checksIn(findings))
	}
	if !strings.Contains(findings[0].What, "No atlas.config") {
		t.Errorf("what-line should name the missing file, got: %q", findings[0].What)
	}
}

func TestRunEmptyMcpsDeclaresNoSecrets(t *testing.T) {
	root := gitRepo(t, ".env\n")
	writeFile(t, root, "atlas.config", "schema_version: 1\nmcps: {}\n")
	mapEntry(t, root, "architecture", "loop", referenceNow)

	if f := Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow}); len(f) != 0 {
		t.Fatalf("an empty mcps map is legal and declares no secrets, got: %v", f)
	}
}

// A secret can arrive from the process environment instead of .env — that is
// how CI and exported shell secrets work, and doctor must not demand a file.
func TestRunAcceptsASecretFromTheProcessEnvironment(t *testing.T) {
	root := gitRepo(t, ".env\n")
	writeFile(t, root, "atlas.config", `schema_version: 1
mcps:
  postgres:
    slot: build
    env:
      - ATLAS_TEST_ONLY_URL
`)
	mapEntry(t, root, "architecture", "loop", referenceNow)

	t.Setenv("ATLAS_TEST_ONLY_URL", "postgres://ci/db")
	if f := Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow}); len(f) != 0 {
		t.Fatalf("a secret set in the process environment must satisfy the check, got: %v", f)
	}
}

func TestRunReportsAMissingSecretByNameAndMcp(t *testing.T) {
	root := gitRepo(t, ".env\n")
	writeFile(t, root, "atlas.config", `schema_version: 1
mcps:
  postgres:
    slot: build
    env:
      - ATLAS_TEST_ONLY_ABSENT
`)
	mapEntry(t, root, "architecture", "loop", referenceNow)

	findings := Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow})
	if len(findings) != 1 {
		t.Fatalf("want exactly 1 finding, got %d: %v", len(findings), checksIn(findings))
	}
	if !strings.Contains(findings[0].What, "ATLAS_TEST_ONLY_ABSENT") ||
		!strings.Contains(findings[0].What, "postgres") {
		t.Errorf("the finding must name both the variable and the MCP that declares it, got: %q", findings[0].What)
	}
}

// Doctor diagnoses only. Nothing in a run may write, move or delete a file —
// v1 explicitly refuses an auto-fixing meta-skill.
func TestRunNeverModifiesTheRepo(t *testing.T) {
	root := brokenRepo(t)

	before := snapshot(t, root)
	Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow})
	after := snapshot(t, root)

	if len(before) != len(after) {
		t.Fatalf("doctor changed the file set: %d entries before, %d after", len(before), len(after))
	}
	for path, mod := range before {
		if after[path] != mod {
			t.Errorf("doctor modified %s", path)
		}
	}
}

// snapshot records every path under root (excluding .git, which git itself
// may touch) with its modification time.
func snapshot(t *testing.T, root string) map[string]time.Time {
	t.Helper()
	out := map[string]time.Time{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		out[rel] = info.ModTime()
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return out
}

func TestRunUnreadableEnvFileIsReportedNotSwallowed(t *testing.T) {
	root := gitRepo(t, ".env\n")
	writeFile(t, root, "atlas.config", "schema_version: 1\nmcps: {}\n")
	mapEntry(t, root, "architecture", "loop", referenceNow)
	// A directory named .env makes the read fail portably.
	if err := os.Mkdir(filepath.Join(root, ".env"), 0o755); err != nil {
		t.Fatalf("mkdir .env: %v", err)
	}

	findings := Run(Options{Dir: root, Schemas: realSchemas(t), Now: referenceNow})
	var reported bool
	for _, f := range findings {
		if strings.Contains(f.What, "Could not read .env") {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("an unreadable .env must be reported — every secret in it would otherwise look missing. Got: %v", findings)
	}
}

// Sanity: the git binary this suite depends on behaves as the code assumes.
func TestGitCheckIgnoreExitCodesAreWhatTheCodeAssumes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	run := func(dir string) int {
		cmd := exec.Command("git", "check-ignore", "-q", ".env")
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ExitCode()
			}
			t.Fatalf("git check-ignore: %v", err)
		}
		return 0
	}
	if got := run(gitRepo(t, ".env\n")); got != 0 {
		t.Errorf("ignored .env should exit 0, got %d", got)
	}
	if got := run(gitRepo(t, "*.log\n")); got != 1 {
		t.Errorf("unignored .env should exit 1, got %d", got)
	}
	if got := run(t.TempDir()); got != 128 {
		t.Errorf("outside a git repo should exit 128, got %d", got)
	}
}
