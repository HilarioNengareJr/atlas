package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckP10 walks map/<type>/<id>.yaml and flags any entry whose `id` field
// disagrees with its own filename (spec/PUNTS.md P10) — a filesystem-level
// property no per-file schema check can see (renaming a file still
// validates cleanly). mapDir is normally "<repo root>/map"; a missing map/
// directory is not an error — there's simply nothing to check yet.
//
// Every directory-type decision here goes through os.Stat (which follows
// symlinks), never a DirEntry's own IsDir() (which does NOT — a symlinked
// directory reports as a non-directory via DirEntry, which used to make an
// entire symlinked type-directory silently invisible to this check; found
// during the 2026-07-21 break session). Anything this function can't read —
// a broken symlink, a permission error, an unexpected directory where an
// entry file was expected — becomes a Finding, never a hard error that
// aborts the rest of the walk: one bad path must not hide every other Map
// entry from being checked (the same session's break finding on
// permission-denied directories aborting the whole validate run).
func CheckP10(mapDir string) ([]Finding, error) {
	typeDirs, err := os.ReadDir(mapDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var findings []Finding
	for _, typeDir := range typeDirs {
		typePath := filepath.Join(mapDir, typeDir.Name())
		info, err := os.Stat(typePath) // follows symlinks, unlike typeDir.IsDir()
		if err != nil {
			findings = append(findings, unreadablePathFinding(typePath, err))
			continue
		}
		if !info.IsDir() {
			continue // a stray file directly under map/ — not this check's job
		}

		files, err := os.ReadDir(typePath)
		if err != nil {
			findings = append(findings, unreadablePathFinding(typePath, err))
			continue
		}
		for _, f := range files {
			full := filepath.Join(typePath, f.Name())
			info, err := os.Stat(full) // follows symlinks, unlike f.IsDir()
			if err != nil {
				findings = append(findings, unreadablePathFinding(full, err))
				continue
			}
			if info.IsDir() {
				// Nesting one level deeper than map/<type>/<id>.yaml expects
				// — whether a real subdirectory (a stray mkdir, a bad
				// copy-paste) or a symlinked one. Either way, anything
				// inside it would otherwise be silently invisible to every
				// check in this package — flag it rather than skip it.
				findings = append(findings, Finding{
					Check:   "p10",
					File:    full,
					Subject: fmt.Sprintf("unexpected directory under %s", typePath),
					What:    "Found a directory here; map/<type>/<id>.yaml expects entry files directly, nothing nested deeper.",
					Why:     "Nothing in this package recurses past this level — any Map entry inside this directory is invisible to every check, not just this one.",
					Next:    "Move the entry file(s) up to be direct children of the type directory, or remove the stray directory.",
				})
				continue
			}

			name := f.Name()
			var base string
			switch {
			case strings.HasSuffix(name, ".yaml"):
				base = strings.TrimSuffix(name, ".yaml")
			case strings.HasSuffix(name, ".yml"):
				base = strings.TrimSuffix(name, ".yml")
			default:
				continue
			}

			doc, err := decodeYAMLFile(full)
			if err != nil {
				findings = append(findings, unreadablePathFinding(full, err))
				continue
			}
			root, ok := doc.(map[string]any)
			if !ok {
				continue // malformed entry is the schema check's job to report
			}
			id, _ := root["id"].(string)
			if id == "" || id == base {
				continue
			}
			findings = append(findings, Finding{
				Check:   "p10",
				File:    full,
				Subject: fmt.Sprintf("id %q vs filename %q", id, name),
				What:    fmt.Sprintf("This entry's id field (%q) does not match its filename (%q).", id, name),
				Why: "Plan citations point at a Map entry by id, not by filename — map.schema.yaml's own comment " +
					"says id \"matches the filename\", but nothing enforces that per-file, so a rename silently " +
					"breaks every citation into this entry.",
				Next: fmt.Sprintf("Rename the file to match id %q, or fix the id field to match the actual filename.", id),
			})
		}
	}
	return findings, nil
}

// unreadablePathFinding turns a broken-symlink/permission/decode error at
// one path into a Finding rather than an error that would abort the whole
// walk — one bad path degrading the check for everything else it should
// have covered is worse than reporting it and moving on.
func unreadablePathFinding(path string, err error) Finding {
	return Finding{
		Check:   "p10",
		File:    path,
		Subject: "unreadable path",
		What:    fmt.Sprintf("Could not read this path: %v", err),
		Why:     "atlas validate couldn't check this Map entry, which means it may be silently excluded from validation rather than actually verified.",
		Next:    "Check permissions on this path, or that a symlink here isn't broken.",
	}
}
