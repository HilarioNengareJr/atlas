// Command atlas is Atlas's single Go binary. Today it implements one
// subcommand, validate (M4.1) — the rest of the seven-command set (init,
// plan, build, verify, sync, help, doctor) lands in later M4 tasks.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "validate" {
		fmt.Fprintln(os.Stderr, "usage: atlas validate [--json] [--schema=plan|records] [path]")
		os.Exit(2)
	}
	os.Exit(runValidate(os.Args[2:]))
}

// runValidate is the whole validate subcommand, split out from main so it
// returns an exit code instead of calling os.Exit directly — keeps this
// file thin (flags, path-type detection, wiring) while every actual check
// lives in internal/validate, per the plan.
func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "emit findings as JSON instead of human text")
	schemaOverride := fs.String("schema", "", "manifest|map|config|plan|records — required for plan/records (spec/PUNTS.md P11) or any fixture outside its real conventional location")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	target := "."
	if fs.NArg() > 0 {
		target = fs.Arg(0)
	}

	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	// --schema is an override for a SINGLE otherwise-undispatchable file
	// (a plan/records instance, or a fixture outside its real conventional
	// location). It has no coherent meaning against a directory: every file
	// there dispatches by its own convention, and forcing them all to one
	// schema silently mis-validates real manifests/Map entries as if they
	// were whatever kind --schema names. Reject the combination outright
	// rather than let it corrupt every finding in the run.
	if info.IsDir() && *schemaOverride != "" {
		fmt.Fprintln(os.Stderr, "--schema is only valid against a single file, not a directory")
		return 2
	}

	var findings []validate.Finding
	if info.IsDir() {
		findings, err = validateDirectory(target)
	} else {
		findings, err = validateFile(target, *schemaOverride)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if *jsonOut {
		out, err := validate.RenderJSON(findings)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Println(string(out))
	} else {
		fmt.Print(validate.RenderHuman(findings))
	}

	if len(findings) > 0 {
		return 1
	}
	return 0
}

// validateFile runs the schema check on one instance, plus any check that's
// self-contained within that one document (P9, when it's a plan) — see the
// plan's language section on cross-file vs. cross-item checks.
func validateFile(path, schemaOverride string) ([]validate.Finding, error) {
	specDir, err := findSpecDir(path)
	if err != nil {
		return nil, err
	}
	schemas, err := validate.LoadSchemas(specDir)
	if err != nil {
		return nil, err
	}

	findings, err := schemas.CheckInstance(path, schemaOverride)
	if err != nil {
		return nil, err
	}

	kind, err := validate.DispatchKind(path, schemaOverride)
	if err == nil && kind == validate.KindPlan {
		doc, derr := decodeForP9(path)
		if derr == nil {
			findings = append(findings, validate.CheckP9(path, doc)...)
		}
	}
	return findings, nil
}

// validateDirectory treats target as the repo root — every instance found
// gets the schema check, plus every cross-file check (ownership matrix, P8,
// P10). A directory argument that isn't actually a repo root (no spec/,
// skills/, map/) simply finds nothing to check under those checks; the
// schema check still runs on whatever instances are recognized.
//
// Every discovered file's Kind is resolved ONCE, by directory convention, at
// discovery time (findInstances) — and that resolved Kind is what gets used
// for both the schema check and the P9 gate below, never re-dispatched.
// (Re-dispatching with the CLI's --schema value here was a real bug: it
// forced every file in the directory to validate against whatever --schema
// named, silently mis-checking real manifests as invalid plans. --schema is
// now rejected outright for directory targets in runValidate, so this
// function never receives one — but resolving Kind once and reusing it,
// rather than calling DispatchKind a second time, is the correct shape
// regardless.)
//
// P9 does not currently run here for any plan/records file: findInstances
// can only discover files by directory convention (manifest/map/config),
// and plan/records have none yet (spec/PUNTS.md P11) — there is no way for
// a directory sweep to recognize an arbitrary file as a plan without being
// told. P9 still runs correctly in single-file mode (validateFile), which
// is what the plan requires; extending it to directory mode is blocked on
// P11, not on anything this function could do differently today.
func validateDirectory(root string) ([]validate.Finding, error) {
	specDir, err := findSpecDir(root)
	if err != nil {
		return nil, err
	}
	schemas, err := validate.LoadSchemas(specDir)
	if err != nil {
		return nil, err
	}

	var findings []validate.Finding

	instances, discoveryFindings, err := findInstances(root)
	if err != nil {
		return nil, err
	}
	findings = append(findings, discoveryFindings...)
	for _, inst := range instances {
		f, err := schemas.CheckInstanceAsKind(inst.path, inst.kind)
		if err != nil {
			findings = append(findings, validate.Finding{
				Check: "schema",
				File:  inst.path,
				What:  err.Error(),
				Why:   "atlas validate couldn't read or check this file.",
				Next:  "See the error above.",
			})
			continue
		}
		findings = append(findings, f...)
	}

	p8, err := validate.CheckP8(specDir)
	if err != nil {
		return nil, err
	}
	findings = append(findings, p8...)

	mapDir := filepath.Join(root, "map")
	p10, err := validate.CheckP10(mapDir)
	if err != nil {
		return nil, err
	}
	findings = append(findings, p10...)

	manifestPaths, manifestFindings, err := findManifests(filepath.Join(root, "skills"))
	if err != nil {
		return nil, err
	}
	findings = append(findings, manifestFindings...)
	entryTypes, err := canonicalEntryTypes(specDir)
	if err != nil {
		return nil, err
	}
	ownership, err := validate.CheckOwnership(manifestPaths, entryTypes)
	if err != nil {
		return nil, err
	}
	findings = append(findings, ownership...)

	return findings, nil
}

// findSpecDir walks upward from start looking for a directory containing
// spec/manifest.schema.yaml, so a single-file target anywhere in the repo
// (not just the root) still finds the schemas to validate against. Bounded
// to 10 levels — a real repo is never nested deeper than that below its spec/.
func findSpecDir(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	dir := abs
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		dir = filepath.Dir(abs)
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "spec", "manifest.schema.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Join(dir, "spec"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("could not find spec/manifest.schema.yaml above " + start + " — is this inside an Atlas repo?")
}

// discoveredInstance pairs a found file with the Kind directory-convention
// dispatch resolved for it — resolved once, here, and reused everywhere
// else so no caller re-dispatches (see validateDirectory's doc comment).
type discoveredInstance struct {
	path string
	kind validate.Kind
}

// findInstances collects every file under root that dispatch can identify
// by directory convention (manifest, map) plus atlas.config — plan/records
// instances have no convention yet (P11) so a directory sweep can't
// discover them; they're only reachable via an explicit --schema on a
// single-file target.
//
// Walks by hand rather than filepath.Walk, for two reasons found during the
// 2026-07-21 break session: (1) filepath.Walk never follows directory
// symlinks (it Lstats), which used to make a symlinked skill or Map
// directory — and everything inside it — completely invisible with no
// warning; this walk resolves every entry via os.Stat (which follows
// symlinks) instead. (2) filepath.Walk's default WalkFunc aborts the ENTIRE
// walk on the first error (e.g. one permission-denied directory used to
// kill validation of the whole rest of the repo); here, an unreadable path
// becomes a Finding and the walk continues. A visited-realpath set guards
// against symlink cycles.
func findInstances(root string) ([]discoveredInstance, []validate.Finding, error) {
	var out []discoveredInstance
	var findings []validate.Finding
	visited := map[string]bool{}

	var walk func(dir string) error
	walk = func(dir string) error {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			findings = append(findings, unreadablePathFinding(dir, err))
			return nil
		}
		if visited[real] {
			return nil
		}
		visited[real] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			findings = append(findings, unreadablePathFinding(dir, err))
			return nil
		}
		for _, e := range entries {
			full := filepath.Join(dir, e.Name())
			info, err := os.Stat(full) // follows symlinks, unlike e.IsDir()
			if err != nil {
				findings = append(findings, unreadablePathFinding(full, err))
				continue
			}
			if info.IsDir() {
				if err := walk(full); err != nil {
					return err
				}
				continue
			}
			if kind, derr := validate.DispatchKind(full, ""); derr == nil {
				out = append(out, discoveredInstance{path: full, kind: kind})
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, nil, err
	}
	return out, findings, nil
}

// findManifests collects every manifest.yaml under skillsDir. Absence of
// skills/ (nothing installed yet, e.g. this repo today) is not an error.
// Directory-ness is decided via os.Stat (follows symlinks), not a
// DirEntry's own IsDir() (which does not) — a symlinked skill directory
// used to be silently skipped entirely, manifest and all (2026-07-21 break
// session).
func findManifests(skillsDir string) ([]string, []validate.Finding, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var out []string
	var findings []validate.Finding
	for _, e := range entries {
		full := filepath.Join(skillsDir, e.Name())
		info, err := os.Stat(full)
		if err != nil {
			findings = append(findings, unreadablePathFinding(full, err))
			continue
		}
		if !info.IsDir() {
			continue
		}
		for _, name := range []string{"manifest.yaml", "manifest.yml"} {
			p := filepath.Join(full, name)
			if _, err := os.Stat(p); err == nil {
				out = append(out, p)
				break
			}
		}
	}
	return out, findings, nil
}

// unreadablePathFinding turns a broken-symlink/permission error at one path
// into a Finding rather than an error that would abort the whole discovery
// walk — mirrors internal/validate's Finding of the same purpose (P10 has
// its own copy since it's a different package; kept as a small, deliberate
// duplication rather than exporting a helper for one two-line function).
func unreadablePathFinding(path string, err error) validate.Finding {
	return validate.Finding{
		Check:   "schema",
		File:    path,
		Subject: "unreadable path",
		What:    fmt.Sprintf("Could not read this path: %v", err),
		Why:     "atlas validate couldn't check this path, which means it may be silently excluded from validation rather than actually verified.",
		Next:    "Check permissions on this path, or that a symlink here isn't broken.",
	}
}

// canonicalEntryTypes reads map.schema.yaml's properties.type.enum — the
// canonical map-entry-type list per that schema's own comment — for the
// ownership matrix to check completeness against.
func canonicalEntryTypes(specDir string) ([]string, error) {
	return validate.MapEntryTypeEnum(specDir)
}

// decodeForP9 is a thin YAML decode for the one caller in this file that
// needs the raw document (P9 operates on decoded data, not a file path).
func decodeForP9(path string) (any, error) {
	return validate.DecodeYAMLFile(path)
}
