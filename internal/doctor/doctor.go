// Package doctor is atlas doctor's engine — the health check for an
// Atlas-configured repo (docs/atlas-plan.md line 150). It answers three
// questions: is the configuration valid and are the secrets it names actually
// present, is .env genuinely gitignored, and is the Map still fresh.
//
// It diagnoses only. Nothing in this package writes a file, edits .gitignore,
// or touches the Map — v1 explicitly refuses an auto-fixing meta-skill
// (project-overview.md, Features Out of Scope), and `atlas init` is the
// command that acts on what doctor finds.
//
// Findings render in the same three-part standard atlas validate uses
// (code-standards.md, Error Handling): what's wrong, why Atlas cares, what to
// do next.
package doctor

import (
	"path/filepath"
	"time"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// Options is one doctor run.
type Options struct {
	// Dir is the repo root being checked — where atlas.config, .env and
	// map/ are expected to live.
	Dir string
	// Schemas is Atlas's compiled spec/, or nil when it isn't reachable from
	// Dir — the normal case on a real project repo (spec/PUNTS.md P14).
	Schemas *validate.SchemaSet
	// SchemasUnavailable says why Schemas is nil, phrased to drop into a
	// Finding's what-line. Ignored when Schemas is non-nil.
	SchemasUnavailable string
	// Now is the instant staleness is measured against. Injected rather than
	// read inside, so tests use real fixture files with fixed dates and stay
	// deterministic without a mock.
	Now time.Time
}

// Run executes every check and returns the findings in a fixed order:
// configuration and secrets, then .env's gitignore status, then Map
// staleness. The order is the order a broken setup should be fixed in — a
// missing config makes the rest meaningless, and a leaked secret matters more
// than a stale Map entry.
//
// A run with no findings returns nil. Every check runs regardless of what the
// ones before it found: doctor's whole job is a complete picture of the
// repo's health, and stopping at the first problem would hide the others.
func Run(opts Options) []validate.Finding {
	var findings []validate.Finding

	env, err := readEnvFile(filepath.Join(opts.Dir, ".env"))
	if err != nil {
		findings = append(findings, validate.Finding{
			Check:   "secrets",
			Subject: filepath.Join(opts.Dir, ".env"),
			What:    "Could not read .env: " + err.Error() + ".",
			Why:     "Doctor resolves declared secrets from .env before falling back to the process environment. Unreadable, every variable that lives only in this file looks missing, so the findings below may overstate the problem.",
			Next:    "Check the file's permissions and that .env is a regular file, then re-run `atlas doctor`.",
		})
		env = envFile{}
	}

	findings = append(findings, CheckConfig(opts.Dir, opts.Schemas, opts.SchemasUnavailable, env)...)
	findings = append(findings, CheckEnvIgnored(opts.Dir)...)
	findings = append(findings, CheckStaleness(filepath.Join(opts.Dir, "map"), opts.Now)...)

	return findings
}
