package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeRepoFiles lays an atlas.config and one Map entry into dir.
func writeRepoFiles(t *testing.T, dir, config, lastConfirmed string) {
	t.Helper()
	write := func(rel, content string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("atlas.config", config)
	write("map/architecture/loop.yaml", `schema_version: 1
id: loop
type: architecture
summary: The lifecycle loop, as built.
created_by_task: t1
last_confirmed: "`+lastConfirmed+`"
`)
}

// cleanFixture builds a repo in which every doctor check genuinely passes.
//
// It lives in a scratch subdirectory of the real repo tree, the same
// convention M4.1's tests use, for two reasons that both matter here:
// findSpecDir walks upward and so resolves Atlas's real spec/ (an unrelated
// t.TempDir() never can — spec/PUNTS.md P14, which is exactly why the
// degraded path exists), and this repo's own .gitignore really does ignore
// .env, so the gitignore check passes against real git and a real ignore
// rule rather than a constructed one.
func cleanFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH; atlas doctor's gitignore check has nothing to ask")
	}
	dir := filepath.Join(repoRoot(t), "cmd_test_scratch_doctor_"+strings.ReplaceAll(t.Name(), "/", "_"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir scratch: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	writeRepoFiles(t, dir, healthyConfig, farFuture)
	return dir
}

// brokenFixture builds a standalone git repository whose .gitignore does NOT
// cover .env and whose Map entry is years old. It has to be a separate repo
// (not a scratch dir inside this one) precisely because this repo ignores
// .env correctly — the unignored case cannot be expressed inside it.
func brokenFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	writeRepoFiles(t, dir, healthyConfig, longPast)
	return dir
}

const healthyConfig = "schema_version: 1\nmcps: {}\n"

// farFuture keeps a fixture fresh against the real clock runDoctor uses.
// runDoctor calls time.Now() by design (the injectable seam is one layer
// down, in doctor.Run) — so a CLI-level fixture has to be dated such that
// the answer can't drift, and a confirmation date that never goes stale is
// the honest way to do that at this layer.
const farFuture = "2999-01-01T00:00:00Z"

const longPast = "2000-01-01T00:00:00Z"

func TestRunDoctorCleanRepoExitsZero(t *testing.T) {
	dir := cleanFixture(t)
	out, code := captureStdout(t, func() int { return runDoctor([]string{dir}) })
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 for a healthy repo. Output:\n%s", code, out)
	}
	if !strings.Contains(out, "atlas doctor: clean") {
		t.Errorf("a clean run must say so explicitly — silence on success is not a report. Got:\n%s", out)
	}
}

func TestRunDoctorFindingsExitOne(t *testing.T) {
	dir := brokenFixture(t)
	out, code := captureStdout(t, func() int { return runDoctor([]string{dir}) })
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 when there are findings. Output:\n%s", code, out)
	}
	if !strings.Contains(out, "does not ignore .env") {
		t.Errorf("expected the unignored-.env finding in the output, got:\n%s", out)
	}
	if !strings.Contains(out, "freshness window") {
		t.Errorf("expected the stale-entry finding in the output, got:\n%s", out)
	}
}

func TestRunDoctorMissingDirectoryExitsTwo(t *testing.T) {
	if code := runDoctor([]string{filepath.Join(t.TempDir(), "nope")}); code != 2 {
		t.Fatalf("exit code = %d, want 2 for a path that doesn't exist", code)
	}
}

func TestRunDoctorFileTargetExitsTwo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atlas.config")
	if err := os.WriteFile(path, []byte(healthyConfig), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if code := runDoctor([]string{path}); code != 2 {
		t.Fatalf("exit code = %d, want 2 — doctor checks a repo, not a single file", code)
	}
}

func TestRunDoctorJSONIsValidAndCarriesTheSameFindings(t *testing.T) {
	dir := brokenFixture(t)
	out, code := captureStdout(t, func() int { return runDoctor([]string{"--json", dir}) })
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	var doc struct {
		Clean    bool `json:"clean"`
		Findings []struct {
			Check string `json:"check"`
			What  string `json:"what"`
			Why   string `json:"why"`
			Next  string `json:"next"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("--json output is not valid JSON: %v\n%s", err, out)
	}
	if doc.Clean {
		t.Error("clean = true despite findings")
	}
	if len(doc.Findings) == 0 {
		t.Fatal("--json must carry the findings, not a thinner shape than the human render")
	}
	for _, f := range doc.Findings {
		if f.What == "" || f.Why == "" || f.Next == "" {
			t.Errorf("JSON finding missing part of the three-part standard: %+v", f)
		}
	}
}

func TestRunDoctorUnknownFlagExitsTwo(t *testing.T) {
	// flag's own error output goes to stderr; only the exit code is asserted.
	if code := runDoctor([]string{"--nonsense"}); code != 2 {
		t.Fatalf("exit code = %d, want 2 for an unknown flag", code)
	}
}

// atlas validate must behave exactly as it did before doctor was added —
// M4.1's contract is not up for renegotiation by M4.2.
func TestValidateStillWorksAfterTheSubcommandSplit(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "spec", "examples", "manifests", "architect.yaml")
	out, code := captureStdout(t, func() int {
		return runValidate([]string{"--schema=manifest", path})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0. Output:\n%s", code, out)
	}
	if !strings.Contains(out, "atlas validate: clean") {
		t.Errorf("validate's clean line must be unchanged by RenderHumanFor, got:\n%s", out)
	}
}
