package validate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Kind names a schema by the same short name spec/<kind>.schema.yaml uses.
type Kind string

const (
	KindManifest Kind = "manifest"
	KindMap      Kind = "map"
	KindConfig   Kind = "config"
	KindPlan     Kind = "plan"
	KindRecords  Kind = "records"
)

// schemaFiles maps each Kind to its file under spec/.
var schemaFiles = map[Kind]string{
	KindManifest: "manifest.schema.yaml",
	KindMap:      "map.schema.yaml",
	KindConfig:   "config.schema.yaml",
	KindPlan:     "plan.schema.yaml",
	KindRecords:  "records.schema.yaml",
}

// SchemaSet holds all five spec/*.schema.yaml, compiled once.
type SchemaSet struct {
	schemas map[Kind]*jsonschema.Schema
}

// LoadSchemas reads and compiles the five schemas from specDir (normally
// "<repo root>/spec"). Each schema is registered under its own $id
// (AddResource) then compiled by that id — the schemas are deliberately
// self-contained (no cross-file $ref), so this needs no shared root.
func LoadSchemas(specDir string) (*SchemaSet, error) {
	set := &SchemaSet{schemas: map[Kind]*jsonschema.Schema{}}
	c := jsonschema.NewCompiler()
	for kind, name := range schemaFiles {
		path := filepath.Join(specDir, name)
		doc, err := decodeYAMLFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
		docMap, ok := doc.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: top-level document is not an object", name)
		}
		id, ok := docMap["$id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("%s: missing $id", name)
		}
		if err := c.AddResource(id, doc); err != nil {
			return nil, fmt.Errorf("registering %s: %w", name, err)
		}
		sch, err := c.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("compiling %s: %w", name, err)
		}
		set.schemas[kind] = sch
	}
	return set, nil
}

// decodeYAMLNodeFile decodes one file into its yaml.Node tree — unlike
// decodeYAMLFile's `any`, a Node retains source line numbers, which is what
// lets a Finding report `file:line` (the plan's stated format) rather than
// just a JSON-pointer field path.
func decodeYAMLNodeFile(path string) (*yaml.Node, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	var node yaml.Node
	if err := dec.Decode(&node); err != nil && err != io.EOF {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := rejectExtraYAMLDocument(dec, path); err != nil {
		return nil, err
	}
	return &node, nil
}

// yamlNodeAt walks a decoded yaml.Node tree following the same path
// segments a jsonschema InstanceLocation gives (map keys, array indices —
// arrays as decimal strings) and returns the Node at that location, or nil
// if the path can't be followed (e.g. the document doesn't actually have
// the shape the schema expected, which is exactly when a Finding needs this
// least — line lookup degrading to "no line" is fine, the Field path still
// says where).
func yamlNodeAt(root *yaml.Node, path []string) *yaml.Node {
	if root == nil {
		return nil
	}
	node := root
	// yaml.Unmarshal into *yaml.Node always wraps the real content in a
	// DocumentNode — unwrap it once before walking.
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	for _, seg := range path {
		switch node.Kind {
		case yaml.MappingNode:
			found := false
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == seg {
					node = node.Content[i+1]
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(node.Content) {
				return nil
			}
			node = node.Content[idx]
		default:
			return nil
		}
	}
	return node
}

// DecodeYAMLFile is decodeYAMLFile, exported for callers outside this
// package that need one raw decoded document (cmd/atlas's P9 wiring, which
// operates on decoded data rather than a file path).
func DecodeYAMLFile(path string) (any, error) {
	return decodeYAMLFile(path)
}

// MapEntryTypeEnum is extractMapEntryTypeEnum, exported so cmd/atlas can
// read the canonical entry-type list (map.schema.yaml's own copy, per its
// comment) to drive the ownership matrix check without duplicating the
// extraction logic P8 already has.
func MapEntryTypeEnum(specDir string) ([]string, error) {
	return extractMapEntryTypeEnum(specDir)
}

// decodeYAMLFile reads and YAML-decodes one file into an `any` shaped the
// way jsonschema expects (map[string]any / []any / string / int|float64 /
// bool / nil) — proven compatible with this library's numeric handling
// during the M4.1 build probe (see docs/decisions.md 2026-07-21).
func decodeYAMLFile(path string) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	var v any
	// A truly empty file returns io.EOF on the FIRST Decode with no document
	// at all — v stays nil (Go zero value), matching the prior
	// yaml.Unmarshal behavior (decodes to YAML null), which the schema check
	// already handles as "got null, want object" rather than a crash.
	if err := dec.Decode(&v); err != nil && err != io.EOF {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := rejectExtraYAMLDocument(dec, path); err != nil {
		return nil, err
	}
	return v, nil
}

// rejectExtraYAMLDocument reports an error if dec has a second YAML document
// still to decode (found during the 2026-07-21 break session: a stray `---`
// document separator used to be silently ignored — only the first document
// was ever checked, with zero indication anything else was in the file).
func rejectExtraYAMLDocument(dec *yaml.Decoder, path string) error {
	var extra any
	err := dec.Decode(&extra)
	switch err {
	case io.EOF:
		return nil
	case nil:
		return fmt.Errorf("%s: file contains more than one YAML document — atlas validate only checks the first; remove the extra document(s) separated by '---'", path)
	default:
		return fmt.Errorf("%s: %w", path, err)
	}
}

// manifestPath / mapPath implement the directory-convention dispatch stated
// elsewhere in the project (code-standards.md for skills/, map.schema.yaml's
// own comment for map/). The manifest FILENAME within skills/atlas-<name>/
// is not actually stated anywhere — "manifest.yaml" here is an assumption
// (the obvious, low-risk default: one manifest per skill directory, named
// after the schema it conforms to), not a confirmed convention. Flagged in
// the build report; cheap to correct later since nothing depends on it yet
// (skills/ is still empty — M3 was skipped).
var (
	manifestPath = regexp.MustCompile(`(^|/)skills/[^/]+/manifest\.ya?ml$`)
	mapPath      = regexp.MustCompile(`(^|/)map/[^/]+/[^/]+\.ya?ml$`)
)

// validKindNames accepts any of the five schema kinds as an explicit
// --schema override, not just plan/records — plan/records NEED it (no
// location convention exists at all, P11), but manifest/map/config also
// benefit from it for fixtures/examples that legitimately live outside
// their real installed-project location (e.g. spec/examples/manifests/,
// used to verify this tool against the M1 retrofit test manifests).
var validKindNames = map[string]Kind{
	"manifest": KindManifest,
	"map":      KindMap,
	"config":   KindConfig,
	"plan":     KindPlan,
	"records":  KindRecords,
}

// DispatchKind determines which schema an instance path validates against.
// Manifest/Map/config dispatch by directory convention first. An explicit
// --schema always wins when given (covers plan/records, which have no
// convention at all per spec/PUNTS.md P11, and covers fixtures/examples of
// any kind that don't live at their real conventional location).
func DispatchKind(path, schemaOverride string) (Kind, error) {
	if schemaOverride != "" {
		if k, ok := validKindNames[schemaOverride]; ok {
			return k, nil
		}
		return "", fmt.Errorf("%s: unknown --schema value %q (want one of manifest, map, config, plan, records)", path, schemaOverride)
	}

	clean := filepath.ToSlash(path)
	switch {
	case manifestPath.MatchString(clean):
		return KindManifest, nil
	case mapPath.MatchString(clean):
		return KindMap, nil
	case filepath.Base(clean) == "atlas.config":
		return KindConfig, nil
	}
	return "", fmt.Errorf(
		"%s: can't tell which schema this is — pass --schema=manifest|map|config|plan|records (plan/records have no location convention yet, spec/PUNTS.md P11)",
		path)
}

// CheckInstance dispatches path to its schema (directory convention or
// schemaOverride) and runs the schema check. A dispatch or decode failure is
// returned as an error, not a Finding — those are usage/IO problems, not
// things the three-part standard applies to.
func (s *SchemaSet) CheckInstance(path, schemaOverride string) ([]Finding, error) {
	kind, err := DispatchKind(path, schemaOverride)
	if err != nil {
		return nil, err
	}
	return s.CheckInstanceAsKind(path, kind)
}

// CheckInstanceAsKind runs the schema check for path against an ALREADY
// RESOLVED kind — no dispatch happens here. Directory-mode callers
// (cmd/atlas's validateDirectory) resolve each file's kind once at
// discovery time and must reuse it here rather than calling DispatchKind
// again with the CLI's --schema value: doing so was a real bug (found in
// review) that force-redispatched every file in a directory sweep against
// whatever --schema named, silently mis-validating real manifests as
// invalid plans.
func (s *SchemaSet) CheckInstanceAsKind(path string, kind Kind) ([]Finding, error) {
	sch, ok := s.schemas[kind]
	if !ok {
		return nil, fmt.Errorf("internal error: no compiled schema for kind %q", kind)
	}
	doc, err := decodeYAMLFile(path)
	if err != nil {
		return nil, err
	}
	// Line numbers are best-effort: a node-tree decode failure (unlikely,
	// since decodeYAMLFile above already succeeded on the same bytes)
	// degrades to no line number rather than failing the whole check.
	node, _ := decodeYAMLNodeFile(path)
	verr := sch.Validate(doc)
	return schemaFindings(path, node, verr), nil
}
