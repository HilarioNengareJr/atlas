package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/HilarioNengareJr/atlas/internal/doctor"
	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// runDoctor is the whole doctor subcommand: flags, schema resolution, wiring
// and exit codes. Every actual check lives in internal/doctor, matching the
// split validate already uses.
//
// Exit codes match validate exactly, because a user should never have to
// remember which Atlas command means what: 0 clean, 1 findings, 2 usage or
// I/O error.
func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "emit findings as JSON instead of human text")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "what: %s is a file, not a directory.\n", dir)
		fmt.Fprintln(os.Stderr, "why:  atlas doctor checks a whole repo — its config, its .env and its Map — not one file.")
		fmt.Fprintln(os.Stderr, "next: point doctor at the directory holding your atlas.config, or run it with no argument to check the current one.")
		return 2
	}

	// Atlas's schemas live in the Atlas repo's spec/, not in this binary
	// (spec/PUNTS.md P14), so a real project repo has none to find. That is
	// expected, not fatal: doctor reports the gap as one finding and runs
	// every check that doesn't need a schema. Aborting here would skip the
	// secrets and gitignore checks, which is the opposite of the point.
	var schemas *validate.SchemaSet
	var unavailable string
	if specDir, err := findSpecDir(dir); err == nil {
		schemas, err = validate.LoadSchemas(specDir)
		if err != nil {
			schemas = nil
			unavailable = "Atlas's schemas could not be loaded: " + err.Error()
		}
	} else {
		unavailable = "Atlas's spec/ directory isn't reachable from here"
	}

	findings := doctor.Run(doctor.Options{
		Dir:                dir,
		Schemas:            schemas,
		SchemasUnavailable: unavailable,
		Now:                time.Now(),
	})

	if *jsonOut {
		out, err := validate.RenderJSON(findings)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Println(string(out))
	} else {
		fmt.Print(validate.RenderHumanFor("doctor", findings))
	}

	if len(findings) > 0 {
		return 1
	}
	return 0
}
