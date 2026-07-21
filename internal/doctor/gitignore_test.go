package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// gitRepo creates a real git repository in a fresh temp directory, with the
// given .gitignore content (empty means no .gitignore at all). A real repo
// and the real git binary — the whole point of this check is that git, not a
// parser of ours, is the authority.
func gitRepo(t *testing.T, gitignore string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH; CheckEnvIgnored has nothing to ask")
	}
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if gitignore != "" {
		if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
			t.Fatalf("write .gitignore: %v", err)
		}
	}
	return dir
}

func TestCheckEnvIgnoredCleanWhenGitIgnoresIt(t *testing.T) {
	if f := CheckEnvIgnored(gitRepo(t, ".env\n")); len(f) != 0 {
		t.Fatalf("an ignored .env is the healthy state and must produce no findings, got: %v", f)
	}
}

func TestCheckEnvIgnoredFlagsAnUnignoredEnv(t *testing.T) {
	findings := CheckEnvIgnored(gitRepo(t, "*.log\n"))
	if len(findings) != 1 {
		t.Fatalf("want exactly 1 finding for an unignored .env, got %d: %v", len(findings), findings)
	}
	if findings[0].Check != "env-gitignored" {
		t.Errorf("Check = %q, want %q", findings[0].Check, "env-gitignored")
	}
	if !strings.Contains(findings[0].What, "does not ignore .env") {
		t.Errorf("what-line should say plainly that git does not ignore .env, got: %q", findings[0].What)
	}
}

// The case that justifies shelling out to git at all: a hand-written parser
// that stops at the first matching line reports ".env is ignored" here, when
// git applies the LAST matching pattern and leaves the file tracked.
func TestCheckEnvIgnoredHonoursNegationRules(t *testing.T) {
	findings := CheckEnvIgnored(gitRepo(t, ".env\n!.env\n"))
	if len(findings) != 1 {
		t.Fatalf("`.env` followed by `!.env` leaves the file NOT ignored — want 1 finding, got %d: %v", len(findings), findings)
	}
}

func TestCheckEnvIgnoredMentionsAnExistingEnvFile(t *testing.T) {
	dir := gitRepo(t, "*.log\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=x\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	findings := CheckEnvIgnored(dir)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].What, "already exists") {
		t.Errorf("an unignored .env that actually exists is more urgent and the message should say so, got: %q", findings[0].What)
	}
}

func TestCheckEnvIgnoredOutsideAGitRepoReportsRatherThanPasses(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	findings := CheckEnvIgnored(t.TempDir())
	if len(findings) != 1 {
		t.Fatalf("want 1 finding when git can't answer, got %d: %v", len(findings), findings)
	}
	if !strings.Contains(findings[0].What, "Could not determine") {
		t.Errorf("an unanswerable check must report the gap, never imply the secrets are safe. Got: %q", findings[0].What)
	}
}

func TestCheckEnvIgnoredFindingsCarryAllThreeParts(t *testing.T) {
	cases := map[string][]validate.Finding{
		"unignored":  CheckEnvIgnored(gitRepo(t, "*.log\n")),
		"no-git-dir": CheckEnvIgnored(t.TempDir()),
	}
	for name, findings := range cases {
		if len(findings) == 0 {
			t.Fatalf("%s: expected at least one finding to inspect", name)
		}
		for _, f := range findings {
			if f.What == "" || f.Why == "" || f.Next == "" {
				t.Errorf("%s: every finding must carry what/why/next (code-standards.md Error Handling), got %+v", name, f)
			}
		}
	}
}
