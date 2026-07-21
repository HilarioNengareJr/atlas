package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// gitTimeout bounds the check-ignore subprocess. git on a healthy repo
// answers in milliseconds; a hang here would hang the whole doctor run, and
// "doctor never returned" is a worse failure than "doctor couldn't tell".
const gitTimeout = 10 * time.Second

// CheckEnvIgnored answers one question: would git ignore a file named .env at
// the root of dir?
//
// It asks git rather than parsing .gitignore, because parsing is wrong in
// ways that fail toward disaster. Negation rules (`.env` followed by `!.env`
// leaves the file NOT ignored — verified against real git while building
// this), nested .gitignore files, .git/info/exclude and core.excludesFile all
// change the answer, and a parser that misses any of them reports "your
// secrets are safe" about a file git would happily commit. A false negative
// on a secrets check is the one error this command must not make, so the
// authority is git itself.
//
// The check runs whether or not .env currently exists: an unignored .env is a
// trap whether the file is there yet or not. Its presence only changes how
// urgent the finding is, which the message says.
//
// git check-ignore -q exit codes (verified 2026-07-21): 0 = ignored,
// 1 = not ignored, 128 = fatal (typically "not a git repository"). Anything
// else is treated as undetermined rather than guessed at.
func CheckEnvIgnored(dir string) []validate.Finding {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-q", ".env")
	cmd.Dir = dir
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		return nil // exit 0 — ignored, which is the healthy state
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return []validate.Finding{envNotIgnoredFinding(dir)}
		}
		return []validate.Finding{undeterminedFinding(dir, fmt.Sprintf(
			"git check-ignore exited %d: %s", exitErr.ExitCode(), firstLine(stderr.Bytes())))}
	}

	if errors.Is(err, exec.ErrNotFound) {
		return []validate.Finding{undeterminedFinding(dir,
			"the git command isn't on PATH")}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return []validate.Finding{undeterminedFinding(dir, fmt.Sprintf(
			"git check-ignore didn't finish within %s", gitTimeout))}
	}
	return []validate.Finding{undeterminedFinding(dir, err.Error())}
}

// envNotIgnoredFinding is the hard fail: git would track .env.
func envNotIgnoredFinding(dir string) validate.Finding {
	what := "git does not ignore .env in this repo."
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		what = "git does not ignore .env in this repo — and a .env file already exists here."
	}
	return validate.Finding{
		Check:   "env-gitignored",
		Subject: filepath.Join(dir, ".env"),
		What:    what,
		Why:     "Secrets are environment, not configuration (code-standards.md, Config vs. Secrets Usage). An unignored .env is one `git add .` away from committing every key the project holds, and git history is not something you can take back quietly.",
		Next:    "Add `.env` to this repo's .gitignore. If a rule later in the file un-ignores it (a `!.env` negation), remove that rule — git applies the last matching pattern.",
	}
}

// undeterminedFinding covers every case where git couldn't answer. It is
// deliberately a Finding and not a silent pass: "Atlas could not verify your
// secrets are ignored" is information the user needs, and treating it as
// clean would be the same lie a broken parser tells.
func undeterminedFinding(dir, reason string) validate.Finding {
	return validate.Finding{
		Check:   "env-gitignored",
		Subject: filepath.Join(dir, ".env"),
		What:    "Could not determine whether .env is gitignored — " + reason + ".",
		Why:     "Atlas asks git itself whether .env is ignored, because that is the only answer that is actually correct. With no answer, it reports the gap rather than assuming your secrets are safe.",
		Next:    "Check that this directory is inside a git repository and that git is installed, then re-run `atlas doctor`. The Map is git-versioned by design, so an Atlas project is expected to be a git repository.",
	}
}

// firstLine trims captured stderr to its first line for a one-line message.
// git's fatal messages are one line and say the useful thing there ("not a
// git repository..."); anything after it is noise inside a Finding.
func firstLine(b []byte) string {
	s := string(b)
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	if s == "" {
		return "no output"
	}
	return s
}
