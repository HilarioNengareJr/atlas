package validate

import (
	"fmt"
	"path/filepath"
	"sort"
)

// CheckP8 verifies the map-entry-type enum agrees across the three schema
// files that each keep their own copy (spec/PUNTS.md P8): manifest.schema.yaml
// ($defs.map_entry_type.enum), map.schema.yaml (properties.type.enum, the
// canonical copy per its own comment), and records.schema.yaml
// ($defs.context_gap.properties.expected_in, the anyOf branch carrying an
// enum). This checks the schema DEFINITIONS themselves, not any instance.
func CheckP8(specDir string) ([]Finding, error) {
	manifestEnum, err := extractManifestEntryTypeEnum(specDir)
	if err != nil {
		return nil, err
	}
	mapEnum, err := extractMapEntryTypeEnum(specDir)
	if err != nil {
		return nil, err
	}
	recordsEnum, err := extractRecordsExpectedInEnum(specDir)
	if err != nil {
		return nil, err
	}

	if setsAgree(manifestEnum, mapEnum) && setsAgree(manifestEnum, recordsEnum) {
		return nil, nil
	}

	return []Finding{{
		Check:   "p8",
		Subject: "map-entry-type enum (manifest.schema.yaml, map.schema.yaml, records.schema.yaml)",
		What: fmt.Sprintf(
			"The map-entry-type enum disagrees across its three copies: manifest.schema.yaml=%v map.schema.yaml=%v records.schema.yaml=%v",
			sortedCopy(manifestEnum), sortedCopy(mapEnum), sortedCopy(recordsEnum)),
		Why: "map.schema.yaml is the canonical list per its own comment; the other two are hand-kept copies with nothing mechanical enforcing the three-way match (spec/PUNTS.md P8) — a drifted copy silently breaks whichever manifest or record relies on the missing or extra type.",
		Next: "Make manifest.schema.yaml's $defs.map_entry_type.enum and records.schema.yaml's " +
			"expected_in enum branch match map.schema.yaml's properties.type.enum exactly.",
	}}, nil
}

func extractManifestEntryTypeEnum(specDir string) ([]string, error) {
	doc, err := decodeYAMLFile(filepath.Join(specDir, "manifest.schema.yaml"))
	if err != nil {
		return nil, err
	}
	root, _ := doc.(map[string]any)
	defs, _ := root["$defs"].(map[string]any)
	met, _ := defs["map_entry_type"].(map[string]any)
	enum, ok := stringEnum(met["enum"])
	if !ok {
		return nil, fmt.Errorf("manifest.schema.yaml: $defs.map_entry_type.enum not found or not a string list")
	}
	return enum, nil
}

func extractMapEntryTypeEnum(specDir string) ([]string, error) {
	doc, err := decodeYAMLFile(filepath.Join(specDir, "map.schema.yaml"))
	if err != nil {
		return nil, err
	}
	root, _ := doc.(map[string]any)
	props, _ := root["properties"].(map[string]any)
	typeField, _ := props["type"].(map[string]any)
	enum, ok := stringEnum(typeField["enum"])
	if !ok {
		return nil, fmt.Errorf("map.schema.yaml: properties.type.enum not found or not a string list")
	}
	return enum, nil
}

func extractRecordsExpectedInEnum(specDir string) ([]string, error) {
	doc, err := decodeYAMLFile(filepath.Join(specDir, "records.schema.yaml"))
	if err != nil {
		return nil, err
	}
	root, _ := doc.(map[string]any)
	defs, _ := root["$defs"].(map[string]any)
	cg, _ := defs["context_gap"].(map[string]any)
	props, _ := cg["properties"].(map[string]any)
	expectedIn, _ := props["expected_in"].(map[string]any)
	anyOf, ok := expectedIn["anyOf"].([]any)
	if !ok {
		return nil, fmt.Errorf("records.schema.yaml: $defs.context_gap.properties.expected_in.anyOf not found")
	}
	for _, branch := range anyOf {
		bm, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		if enum, ok := stringEnum(bm["enum"]); ok {
			return enum, nil
		}
	}
	return nil, fmt.Errorf("records.schema.yaml: no anyOf branch under expected_in carries an enum")
}

// stringEnum type-asserts a decoded YAML `enum:` list into []string.
func stringEnum(v any) ([]string, bool) {
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// setsAgree compares two string lists as sets — order between the three
// independent schema files isn't a stated requirement, only membership.
func setsAgree(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, x := range a {
		set[x] = true
	}
	for _, x := range b {
		if !set[x] {
			return false
		}
	}
	return true
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
