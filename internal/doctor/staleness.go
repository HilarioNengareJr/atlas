package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/HilarioNengareJr/atlas/internal/validate"
)

// StaleAfter is how long a Map entry may go unconfirmed before doctor calls
// it stale.
//
// Hardcoded on purpose, and punted as spec/PUNTS.md P13: no schema field
// holds a threshold, and adding one to config.schema.yaml for a single
// tunable was scope M4.2 didn't own. 30 days is a first guess that different
// entry types will almost certainly disagree with — `decisions` ages nothing
// like `components` — which is exactly what the punt is for.
const StaleAfter = 30 * 24 * time.Hour

// CheckStaleness walks mapDir (a repo's map/) and reports every entry whose
// last_confirmed is older than StaleAfter, measured against now.
//
// now is a parameter rather than a call to time.Now inside, so tests run
// against real fixture files with fixed dates and stay deterministic without
// a clock interface or a mock — the no-mocks standard M4.1 set.
//
// The walk follows the same map/<type>/<id>.yaml convention validate uses,
// and repeats two lessons from M4.1's break session: directory-ness is
// decided by os.Stat (which follows symlinks) rather than DirEntry.IsDir()
// (which does not, so a symlinked map/ used to be invisible), and an
// unreadable path degrades to its own Finding instead of aborting the walk.
func CheckStaleness(mapDir string, now time.Time) []validate.Finding {
	info, err := os.Stat(mapDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []validate.Finding{{
				Check:   "map-staleness",
				Subject: mapDir,
				What:    "No map/ directory in this repo.",
				Why:     "The Map is the accumulated stock of understanding that every Atlas skill reads and writes — without it, agents on this project start from zero every session, which is the exact problem Atlas exists to solve.",
				Next:    "Run `atlas init` to survey the repo and build an initial Map, then confirm it looks true.",
			}}
		}
		return []validate.Finding{unreadable(mapDir, err)}
	}
	if !info.IsDir() {
		return []validate.Finding{{
			Check:   "map-staleness",
			Subject: mapDir,
			What:    "map/ exists but is not a directory.",
			Why:     "The Map is a directory of files (map/<type>/<id>.yaml), one entry per file, so it stays git-versioned and legible without Atlas installed. A single file here can't hold that.",
			Next:    "Move the file aside and let `atlas init` build a real map/ directory.",
		}}
	}

	var findings []validate.Finding
	for _, path := range mapEntryPaths(mapDir, &findings) {
		findings = append(findings, checkEntry(path, now)...)
	}
	return findings
}

// mapEntryPaths collects every map/<type>/<id>.yaml path under mapDir, in
// sorted order so a run's findings are stable between identical runs.
// Unreadable paths become Findings appended to out rather than errors.
func mapEntryPaths(mapDir string, out *[]validate.Finding) []string {
	typeDirs, err := os.ReadDir(mapDir)
	if err != nil {
		*out = append(*out, unreadable(mapDir, err))
		return nil
	}

	var paths []string
	for _, td := range typeDirs {
		typePath := filepath.Join(mapDir, td.Name())
		info, err := os.Stat(typePath) // follows symlinks, unlike td.IsDir()
		if err != nil {
			*out = append(*out, unreadable(typePath, err))
			continue
		}
		if !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(typePath)
		if err != nil {
			*out = append(*out, unreadable(typePath, err))
			continue
		}
		for _, e := range entries {
			full := filepath.Join(typePath, e.Name())
			ext := filepath.Ext(e.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			fi, err := os.Stat(full)
			if err != nil {
				*out = append(*out, unreadable(full, err))
				continue
			}
			if fi.IsDir() {
				continue
			}
			paths = append(paths, full)
		}
	}
	sort.Strings(paths)
	return paths
}

// checkEntry reads one Map entry and reports it if last_confirmed is missing,
// unparseable, or older than StaleAfter.
//
// A missing or malformed last_confirmed is a Finding here, not a silent skip:
// the field is required by map.schema.yaml, and an entry doctor can't date is
// an entry whose freshness nobody is checking — the same practical outcome as
// a stale one. Explaining the schema violation in full is atlas validate's
// job; not passing over it silently is doctor's.
func checkEntry(path string, now time.Time) []validate.Finding {
	doc, err := validate.DecodeYAMLFile(path)
	if err != nil {
		return []validate.Finding{{
			Check:   "map-staleness",
			Subject: path,
			What:    fmt.Sprintf("Could not read this Map entry: %v", err),
			Why:     "An entry doctor can't read is an entry whose freshness nobody is checking, and skills consuming the Map will hit the same problem mid-task.",
			Next:    "Run `atlas validate` on this file for the full explanation, then fix it.",
		}}
	}

	root, ok := doc.(map[string]any)
	if !ok {
		return []validate.Finding{malformed(path, "it is not a YAML mapping")}
	}
	raw, ok := root["last_confirmed"].(string)
	if !ok || raw == "" {
		return []validate.Finding{malformed(path, "it has no last_confirmed field")}
	}
	confirmed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return []validate.Finding{malformed(path, fmt.Sprintf("last_confirmed %q is not an RFC 3339 timestamp", raw))}
	}

	age := now.Sub(confirmed)
	if age <= StaleAfter {
		return nil
	}
	return []validate.Finding{{
		Check:   "map-staleness",
		Subject: path,
		What: fmt.Sprintf("This Map entry was last confirmed %s ago (%s), past the %s freshness window.",
			roundDays(age), confirmed.Format("2006-01-02"), roundDays(StaleAfter)),
		Why:  "The Map is only worth reading if it is true. A stale entry is worse than a missing one — agents cite it with confidence and act on something the codebase stopped doing weeks ago.",
		Next: "Re-read this entry against the code. If it is still true, update last_confirmed to today; if it isn't, correct it first.",
	}}
}

// malformed is the shared shape for an entry doctor can't date.
func malformed(path, reason string) validate.Finding {
	return validate.Finding{
		Check:   "map-staleness",
		Subject: path,
		What:    "Could not determine this Map entry's age — " + reason + ".",
		Why:     "last_confirmed is required by spec/map.schema.yaml precisely so freshness is a checkable property. Without it, this entry silently escapes every staleness check Atlas runs.",
		Next:    "Run `atlas validate` on this file for the full schema explanation, then add or fix last_confirmed.",
	}
}

// unreadable turns a permission/broken-symlink error into a Finding rather
// than an error that would abort the rest of the walk.
func unreadable(path string, err error) validate.Finding {
	return validate.Finding{
		Check:   "map-staleness",
		Subject: path,
		What:    fmt.Sprintf("Could not read this path: %v", err),
		Why:     "Anything doctor can't read is excluded from the staleness check rather than actually verified — reporting the rest as fresh would overstate what was checked.",
		Next:    "Check permissions on this path, or that a symlink here isn't broken.",
	}
}

// roundDays renders a duration in whole days — the only unit a freshness
// window is ever discussed in.
func roundDays(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
