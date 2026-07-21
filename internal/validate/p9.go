package validate

import (
	"fmt"
	"reflect"
)

// CheckP9 finds plan steps that share an `id` with different content —
// plan.schema.yaml's uniqueItems only rejects byte-identical duplicate
// steps; two DIFFERENT steps sharing an id validate cleanly today (spec/
// PUNTS.md P9). This is a cross-ITEM check within one document, not a
// cross-FILE one, so it runs even when a single plan file is validated on
// its own (see cmd/atlas/validate.go's single-file mode).
func CheckP9(path string, doc any) []Finding {
	root, ok := doc.(map[string]any)
	if !ok {
		return nil // malformed top-level document is the schema check's job to report
	}
	stepsAny, ok := root["steps"].([]any)
	if !ok {
		return nil // missing/malformed `steps` is the schema check's job to report
	}

	seen := map[string]any{}
	reported := map[string]bool{}
	var findings []Finding
	for _, s := range stepsAny {
		step, ok := s.(map[string]any)
		if !ok {
			continue
		}
		id, _ := step["id"].(string)
		if id == "" {
			continue
		}
		prev, exists := seen[id]
		if !exists {
			seen[id] = step
			continue
		}
		if reported[id] || reflect.DeepEqual(prev, step) {
			continue // byte-identical duplicates are already a schema-check failure (uniqueItems)
		}
		reported[id] = true
		findings = append(findings, Finding{
			Check:   "p9",
			File:    path,
			Subject: fmt.Sprintf("step id %q", id),
			What:    fmt.Sprintf("Two steps share id %q with different content.", id),
			Why:     "Decision records cite a plan step by id (records.schema.yaml's `clause`) — if two steps share an id, a citation can't tell which step it actually means.",
			Next:    "Give each step its own unique id, or remove/renumber the duplicate.",
		})
	}
	return findings
}
