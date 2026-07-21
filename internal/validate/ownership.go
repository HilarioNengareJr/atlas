package validate

import "fmt"

// CheckOwnership verifies every Map entry type has at least one maintainer
// across all installed skill manifests (docs/atlas-plan.md line 94: "every
// Map entry type must have at least one skill declaring maintains over it —
// an unmaintained entry type is staleness by design, detectable statically
// by atlas validate"; manifest.schema.yaml's own comment attributes this
// check to atlas validate, not the schema). manifestPaths are the
// manifest.yaml files found under skills/. entryTypes is the canonical list
// (map.schema.yaml's enum, per its own comment) — passed in rather than
// re-derived here so the caller decides which copy is authoritative.
//
// standup-ledger-slot is NOT special-cased here even though it's reserved
// and currently unmaintained by design (spec/PUNTS.md P6) — P6 explicitly
// leaves whether to suppress that as a decision forced at M4.7 (atlas help)
// or the standup scope doc, not M4.1. Silently excluding it here would be
// this check quietly resolving P6 on its own; it flags today, as the punt
// itself says is the expected v1 behavior.
func CheckOwnership(manifestPaths []string, entryTypes []string) ([]Finding, error) {
	maintained := map[string]bool{}
	for _, p := range manifestPaths {
		doc, err := decodeYAMLFile(p)
		if err != nil {
			return nil, err
		}
		root, ok := doc.(map[string]any)
		if !ok {
			continue // malformed manifest is the schema check's job to report
		}
		list, _ := root["maintains"].([]any)
		for _, item := range list {
			if s, ok := item.(string); ok {
				maintained[s] = true
			}
		}
	}

	var findings []Finding
	for _, t := range entryTypes {
		if maintained[t] {
			continue
		}
		findings = append(findings, Finding{
			Check:   "ownership-matrix",
			Subject: fmt.Sprintf("Map entry type %q", t),
			What:    fmt.Sprintf("No installed skill declares %q in its `maintains` list.", t),
			Why:     "An entry type nothing maintains goes stale with no one responsible for keeping it true — staleness by design, not by accident (docs/atlas-plan.md's ownership matrix).",
			Next:    fmt.Sprintf("Add %q to some skill's `maintains` list, or remove the entry type from map.schema.yaml if it's genuinely unused.", t),
		})
	}
	return findings, nil
}
