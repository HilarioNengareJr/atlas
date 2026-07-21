package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// ConfigFile is the one filename an Atlas config lives under. The same
// constant is the directory-convention dispatch rule in internal/validate
// (DispatchKind) — kept here as a named value rather than a bare literal so
// the two places that care are both greppable.
const ConfigFile = "atlas.config"

// CheckConfig runs the config half of atlas doctor against dir: the
// declaration is schema-valid, and every environment variable it names
// actually resolves.
//
// This subsumes what the build plan called the "MCP reachability" check.
// A declared MCP's slot is already constrained to the six-value enum by
// config.schema.yaml, and its secrets are the env check below — so a separate
// third check would have had nothing left to do. Proving an MCP is genuinely
// reachable means opening a connection, which needs an MCP client Go doesn't
// have yet: punted as spec/PUNTS.md P12.
//
// schemas may be nil when Atlas's own spec/ isn't reachable from dir, which
// is the normal case on a real user repo (spec/PUNTS.md P14). That degrades
// the schema half to its own Finding; the env half still runs, because a
// missing secret is worth reporting whether or not the file's shape could be
// verified.
func CheckConfig(dir string, schemas *validate.SchemaSet, schemasUnavailable string, env envFile) []validate.Finding {
	path := filepath.Join(dir, ConfigFile)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return []validate.Finding{{
				Check:   "config",
				Subject: path,
				What:    "No atlas.config in this directory.",
				Why:     "atlas.config is what makes a repo an Atlas project — it declares which MCPs the project uses and which lifecycle slot each one serves. Without it there is no configuration for Atlas to check, and `atlas sync` has nothing to compile.",
				Next:    "Run `atlas init` here to create one, or re-run `atlas doctor` pointed at the directory that holds your atlas.config.",
			}}
		}
		return []validate.Finding{{
			Check:   "config",
			Subject: path,
			What:    fmt.Sprintf("Could not read atlas.config: %v", err),
			Why:     "Atlas can't check a configuration it can't open, and reporting the rest as healthy would imply this file was verified when it wasn't.",
			Next:    "Check the file's permissions and that it is a regular file.",
		}}
	}

	var findings []validate.Finding

	if schemas == nil {
		findings = append(findings, validate.Finding{
			Check:   "config",
			Subject: path,
			What:    "Could not schema-check atlas.config — " + schemasUnavailable + ".",
			Why:     "Atlas's schemas live in the Atlas repo's spec/ directory, not inside the binary yet (spec/PUNTS.md P14), so a project repo has nothing to check against. Every other doctor check still ran — only the shape of this one file is unverified.",
			Next:    "Run `atlas validate` from inside a checkout of the Atlas repo to check this file's shape, or ignore this if you only needed the secrets and staleness checks.",
		})
	} else {
		schemaFindings, err := schemas.CheckInstanceAsKind(path, validate.KindConfig)
		if err != nil {
			return append(findings, validate.Finding{
				Check:   "config",
				Subject: path,
				What:    fmt.Sprintf("Could not parse atlas.config: %v", err),
				Why:     "An unparseable config means every MCP declaration in it is unknown to Atlas — the secrets check below has nothing reliable to read.",
				Next:    "Fix the YAML syntax error above, then re-run `atlas doctor`.",
			})
		}
		findings = append(findings, schemaFindings...)
	}

	findings = append(findings, checkDeclaredEnv(path, env)...)
	return findings
}

// checkDeclaredEnv reads mcps[*].env from the config and reports every named
// variable that doesn't resolve.
//
// It decodes the file independently of the schema check on purpose: on a user
// repo the schema check can't run at all (P14), and a missing API key is
// exactly the thing doctor exists to catch there. It reads defensively rather
// than asserting the schema-guaranteed shape, for the same reason.
func checkDeclaredEnv(path string, env envFile) []validate.Finding {
	doc, err := validate.DecodeYAMLFile(path)
	if err != nil {
		return []validate.Finding{{
			Check:   "secrets",
			Subject: path,
			What:    fmt.Sprintf("Could not read the MCP declarations: %v", err),
			Why:     "Doctor checks secrets by reading which environment variables each declared MCP needs. An unreadable config means no secret could be checked at all.",
			Next:    "Fix the error above, then re-run `atlas doctor`.",
		}}
	}

	root, ok := doc.(map[string]any)
	if !ok {
		return []validate.Finding{{
			Check:   "secrets",
			Subject: path,
			What:    "atlas.config is not a YAML mapping at its top level.",
			Why:     "Doctor expects the declared shape (a `mcps` mapping) to find which environment variables the project needs. A different shape means no secret could be checked.",
			Next:    "Compare the file against spec/config.schema.yaml — the top level is an object with `schema_version` and `mcps`.",
		}}
	}
	mcps, ok := root["mcps"].(map[string]any)
	if !ok {
		return nil // absent or empty mcps declares no secrets — nothing to check
	}

	var findings []validate.Finding
	// Sorted so a run's findings are stable — Go map iteration order is
	// random, and a report that reshuffles between identical runs is one
	// nobody can diff.
	for _, name := range sortedKeys(mcps) {
		entry, ok := mcps[name].(map[string]any)
		if !ok {
			continue
		}
		vars, ok := entry["env"].([]any)
		if !ok {
			continue // an MCP needing no secrets is legal
		}
		for _, v := range vars {
			varName, ok := v.(string)
			if !ok || varName == "" {
				continue
			}
			if env.isSet(varName) {
				continue
			}
			findings = append(findings, validate.Finding{
				Check:   "secrets",
				Subject: fmt.Sprintf("%s → mcps.%s.env: %s", path, name, varName),
				What:    fmt.Sprintf("%s is declared by the %q MCP but is not set.", varName, name),
				Why:     "Configuration is code and secrets are environment (code-standards.md, Config vs. Secrets Usage). atlas.config names the variable; the value has to come from .env or the process environment. Missing, the MCP fails on its first call mid-task rather than here.",
				Next:    fmt.Sprintf("Add `%s=...` to the .env file next to atlas.config, or export it in your environment. .env must stay gitignored.", varName),
			})
		}
	}
	return findings
}

// sortedKeys returns m's keys in sorted order.
func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
