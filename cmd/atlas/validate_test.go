package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written to it, alongside fn's own return value.
func captureStdout(t *testing.T, fn func() int) (string, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	code := fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(out), code
}

// repoRoot resolves the real atlas repo root from cmd/atlas's test working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "spec", "manifest.schema.yaml")); err != nil {
		t.Fatalf("could not find repo root at %s: %v", root, err)
	}
	return root
}

func TestRunValidateRealM1ExamplesCleanExitZero(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{"architect.yaml", "build.yaml", "review.yaml"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "spec", "examples", "manifests", name)
			out, code := captureStdout(t, func() int {
				return runValidate([]string{"--schema=manifest", path})
			})
			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (matches the M4 build-plan exit criterion). Output:\n%s", code, out)
			}
		})
	}
}

func TestRunValidateCorruptedInstanceExitOne(t *testing.T) {
	// A single-file target needs the real spec/ reachable above it, so the
	// fixture lives under a scratch subdirectory of the real repo tree
	// (findSpecDir walks upward from the file) rather than an unrelated
	// t.TempDir(), which findSpecDir could never resolve.
	root := repoRoot(t)
	scratch := filepath.Join(root, "cmd_test_scratch")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatalf("mkdir scratch: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(scratch) })

	path := filepath.Join(scratch, "corrupted.yaml")
	if err := os.WriteFile(path, []byte("schema_version: \"1\"\nname: x\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	out, code := captureStdout(t, func() int {
		return runValidate([]string{"--schema=manifest", path})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1. Output:\n%s", code, out)
	}
}

func TestRunValidateJSONModeIsValidAndClean(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "spec", "examples", "manifests", "architect.yaml")
	out, code := captureStdout(t, func() int {
		return runValidate([]string{"--json", "--schema=manifest", path})
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0. Output:\n%s", code, out)
	}
	if !strings.Contains(out, `"clean": true`) || !strings.Contains(out, `"findings": []`) {
		t.Fatalf("--json output doesn't look like a clean report: %s", out)
	}
}

// Review fix: --schema alongside a directory target used to silently
// force-redispatch every discovered file against whatever --schema named —
// e.g. a real manifest.yaml would be checked against plan.schema.yaml and
// report bogus "missing steps/footprint" findings. It must now be rejected
// outright as a usage error instead.
func TestRunValidateSchemaOverrideRejectedForDirectory(t *testing.T) {
	root := repoRoot(t)
	out, code := captureStdout(t, func() int {
		return runValidate([]string{"--schema=plan", root})
	})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error). Output:\n%s", code, out)
	}
}

// Review fix, end-to-end proof of the actual bug: before the fix, this
// exact scenario (a real manifest inside a directory, validated with
// --schema=plan) produced findings claiming the manifest was missing plan
// fields. Now it must be rejected before any file is even looked at.
func TestRunValidateSchemaOverrideNeverMisdispatchesRealFiles(t *testing.T) {
	root := repoRoot(t)
	fixture := t.TempDir()
	mustCopySpec(t, root, fixture)
	skillDir := filepath.Join(fixture, "skills", "atlas-foo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := "schema_version: 1\nname: atlas-foo\nstage: survey\nconsumes: []\nmaintains: []\nemits: []\nrequires_slots: []\n"
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	out, code := captureStdout(t, func() int {
		return runValidate([]string{"--schema=plan", fixture})
	})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (rejected before touching any file)", code)
	}
	if strings.Contains(out, "missing properties") || strings.Contains(out, "additional properties") {
		t.Fatalf("the real manifest was mis-validated against plan.schema.yaml — bug reproduced. Output:\n%s", out)
	}
}

func TestRunValidateDirectoryModeFindsOwnershipGaps(t *testing.T) {
	// The real repo's skills/ is still empty (M3 was skipped) — every Map
	// entry type is a genuine, expected ownership-matrix finding.
	root := repoRoot(t)
	out, code := captureStdout(t, func() int {
		return runValidate([]string{root})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (the real repo has no skills/ installed yet). Output:\n%s", code, out)
	}
	if !strings.Contains(out, "maintains") {
		t.Fatalf("expected ownership-matrix findings in directory mode, got:\n%s", out)
	}
}

func TestRunValidateAttackCases(t *testing.T) {
	root := repoRoot(t)

	t.Run("P10 renamed Map file", func(t *testing.T) {
		fixture := t.TempDir()
		mustCopySpec(t, root, fixture)
		typeDir := filepath.Join(fixture, "map", "architecture")
		if err := os.MkdirAll(typeDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		content := "schema_version: 1\nid: loop\ntype: architecture\nsummary: t\ncreated_by_task: t1\nlast_confirmed: \"2026-07-21T00:00:00Z\"\n"
		if err := os.WriteFile(filepath.Join(typeDir, "zoop.yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		out, code := captureStdout(t, func() int { return runValidate([]string{fixture}) })
		if code != 1 || !strings.Contains(out, "does not match its filename") {
			t.Fatalf("P10 attack case not caught. exit=%d output:\n%s", code, out)
		}
	})

	t.Run("P9 duplicate step id", func(t *testing.T) {
		fixture := t.TempDir()
		mustCopySpec(t, root, fixture)
		planPath := filepath.Join(fixture, "dup.yaml")
		content := "schema_version: 1\nid: p\ntitle: t\nsteps:\n  - {id: s1, description: a, done_when: a}\n  - {id: s1, description: b, done_when: b}\nfootprint: [\"**\"]\nverification: [\"ok\"]\ncitations: []\n"
		if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		out, code := captureStdout(t, func() int { return runValidate([]string{"--schema=plan", planPath}) })
		if code != 1 || !strings.Contains(out, "share id") {
			t.Fatalf("P9 attack case not caught. exit=%d output:\n%s", code, out)
		}
	})

	t.Run("P8 diverged enum copy", func(t *testing.T) {
		fixture := t.TempDir()
		mustCopySpec(t, root, fixture)
		recordsPath := filepath.Join(fixture, "spec", "records.schema.yaml")
		b, err := os.ReadFile(recordsPath)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		diverged := strings.Replace(string(b), "library-manifest", "library-manifest-DIVERGED", 1)
		if err := os.WriteFile(recordsPath, []byte(diverged), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		out, code := captureStdout(t, func() int { return runValidate([]string{fixture}) })
		if code != 1 || !strings.Contains(out, "disagrees across its three copies") {
			t.Fatalf("P8 attack case not caught. exit=%d output:\n%s", code, out)
		}
	})

	t.Run("ownership matrix gap", func(t *testing.T) {
		fixture := t.TempDir()
		mustCopySpec(t, root, fixture)
		out, code := captureStdout(t, func() int { return runValidate([]string{fixture}) })
		if code != 1 || !strings.Contains(out, "maintains") {
			t.Fatalf("ownership-matrix attack case not caught. exit=%d output:\n%s", code, out)
		}
	})
}

// Break-session regression: a symlinked skill directory under skills/ used
// to be entirely invisible (DirEntry.IsDir() reports false for a symlink),
// so a real manifest declaring `maintains` inside it silently didn't count.
// Fixed 2026-07-21 by resolving via os.Stat, which follows symlinks.
func TestRunValidateFollowsSymlinkedSkillDirectory(t *testing.T) {
	fixture := t.TempDir()
	mustCopySpec(t, repoRoot(t), fixture)
	real := filepath.Join(fixture, "real_elsewhere")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := "schema_version: 1\nname: atlas-shared\nstage: survey\nconsumes: []\nmaintains: [architecture, conventions]\nemits: []\nrequires_slots: []\n"
	if err := os.WriteFile(filepath.Join(real, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	skillsDir := filepath.Join(fixture, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.Symlink(real, filepath.Join(skillsDir, "atlas-shared")); err != nil {
		t.Skipf("symlinks not supported on this filesystem: %v", err)
	}

	out, code := captureStdout(t, func() int { return runValidate([]string{fixture}) })
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (5 remaining unmaintained types). Output:\n%s", code, out)
	}
	if strings.Count(out, "Map entry type") != 5 {
		t.Fatalf("expected exactly 5 ownership-matrix findings (architecture+conventions now maintained via the symlink), got:\n%s", out)
	}
}

// Break-session regression: a permission-denied skill directory used to
// abort validation of the ENTIRE repo. Fixed 2026-07-21 — it must now
// degrade to its own finding while everything else still gets checked.
func TestRunValidatePermissionDeniedDirDoesNotAbortWholeRun(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — permission bits don't block root reads")
	}
	fixture := t.TempDir()
	mustCopySpec(t, repoRoot(t), fixture)
	locked := filepath.Join(fixture, "skills", "atlas-locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(locked, "manifest.yaml"), []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ok := filepath.Join(fixture, "skills", "atlas-ok")
	if err := os.MkdirAll(ok, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	okManifest := "schema_version: 1\nname: atlas-ok\nstage: survey\nconsumes: []\nmaintains: [architecture]\nemits: []\nrequires_slots: []\n"
	if err := os.WriteFile(filepath.Join(ok, "manifest.yaml"), []byte(okManifest), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	out, code := captureStdout(t, func() int { return runValidate([]string{fixture}) })
	if code != 1 {
		t.Fatalf("exit code = %d, want 1. Output:\n%s", code, out)
	}
	if !strings.Contains(out, "permission denied") {
		t.Fatalf("expected a finding about the permission-denied path, got:\n%s", out)
	}
	// atlas-ok's manifest must STILL have been read (architecture no longer
	// unmaintained) — proof the rest of the run wasn't aborted.
	if strings.Count(out, "Map entry type") != 6 {
		t.Fatalf("expected exactly 6 ownership-matrix findings (atlas-ok's manifest still read despite atlas-locked failing), got:\n%s", out)
	}
}

func mustCopySpec(t *testing.T, root, dstRoot string) {
	t.Helper()
	specDst := filepath.Join(dstRoot, "spec")
	if err := os.MkdirAll(specDst, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"manifest.schema.yaml", "map.schema.yaml", "config.schema.yaml", "plan.schema.yaml", "records.schema.yaml"} {
		b, err := os.ReadFile(filepath.Join(root, "spec", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(specDst, name), b, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
