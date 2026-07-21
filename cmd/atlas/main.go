// Command atlas is Atlas's single Go binary. Today it implements two
// subcommands, validate (M4.1) and doctor (M4.2) — the rest of the
// seven-command set (init, plan, build, verify, sync, help) lands in later
// M4 tasks.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// subcommands maps each command name to its implementation. A map rather
// than a chain of string comparisons because five more commands are coming
// and the ceiling is a known seven (docs/atlas-plan.md line 186) — cheap to
// do now, at two.
var subcommands = map[string]func(args []string) int{
	"validate": runValidate,
	"doctor":   runDoctor,
}

func main() {
	if len(os.Args) < 2 {
		usage("")
		os.Exit(2)
	}
	run, ok := subcommands[os.Args[1]]
	if !ok {
		usage(os.Args[1])
		os.Exit(2)
	}
	os.Exit(run(os.Args[2:]))
}

// usage prints the three-part error standard for a bad or missing
// subcommand — the same shape every Atlas interruption uses
// (code-standards.md, Error Handling), not a bare usage line.
func usage(unknown string) {
	names := make([]string, 0, len(subcommands))
	for name := range subcommands {
		names = append(names, name)
	}
	sort.Strings(names)

	if unknown == "" {
		fmt.Fprintln(os.Stderr, "what: no subcommand given.")
	} else {
		fmt.Fprintf(os.Stderr, "what: %q is not an atlas subcommand.\n", unknown)
	}
	fmt.Fprintln(os.Stderr, "why:  atlas does one thing per subcommand, and needs to be told which.")
	fmt.Fprintf(os.Stderr, "next: run one of: %v\n", names)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  atlas validate [--json] [--schema=manifest|map|config|plan|records] [path]")
	fmt.Fprintln(os.Stderr, "  atlas doctor   [--json] [dir]")
}

// findSpecDir walks upward from start looking for a directory containing
// spec/manifest.schema.yaml, so a single-file target anywhere in the repo
// (not just the root) still finds the schemas to validate against. Bounded
// to 10 levels — a real repo is never nested deeper than that below its spec/.
//
// Shared by both subcommands. Note what it cannot do: a real Atlas project
// repo has an atlas.config and a map/ but no spec/ of its own, so this
// returns an error there. validate treats that as fatal; doctor degrades and
// keeps checking (spec/PUNTS.md P14).
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
