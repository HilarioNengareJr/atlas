// Package validate is atlas validate's engine — a type-checker for skills
// (docs/atlas-plan.md line 102), not just a JSON Schema instance checker. It
// runs two categories of check: schema checks (one instance against its
// schema) and cross-file/cross-item checks the project has already named for
// this milestone (the ownership matrix, and P8/P9/P10 from spec/PUNTS.md).
// Every finding renders in the project's three-part error standard
// (code-standards.md Error Handling): what's wrong, why Atlas cares, what to
// do next — never a bare location+message.
package validate

import (
	"encoding/json"
	"fmt"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

// printer is the one message.Printer this package needs. jsonschema's
// ErrorKind.LocalizedString panics on a nil *message.Printer — found while
// proving the library out before writing this package — so every call site
// shares this real one instead of passing nil.
var printer = message.NewPrinter(message.MatchLanguage("en"))

// Finding is one thing atlas validate found wrong. Schema-shape findings
// (Check == "schema") carry File/Field, a real location within one document.
// Cross-file findings (ownership-matrix, p8, p9, p10) have no single anchor
// point — they carry Subject instead, describing what the finding is about.
type Finding struct {
	Check string `json:"check"` // "schema" | "ownership-matrix" | "p8" | "p9" | "p10"
	File  string `json:"file,omitempty"`
	// Line is the source line within File, when it could be resolved from
	// the document's yaml.Node tree — best-effort (see schemaFindings); 0
	// means unavailable, not "line zero".
	Line    int    `json:"line,omitempty"`
	Field   string `json:"field,omitempty"`   // JSON-pointer-ish path within File, e.g. "/consumes/0"
	Subject string `json:"subject,omitempty"` // cross-file findings only
	What    string `json:"what"`
	Why     string `json:"why"`
	Next    string `json:"next"`
}

// location renders whichever of File(:Line)/Field or Subject this Finding
// carries. file:line is preferred (the plan's stated format) when a line
// was resolved; the JSON-pointer Field is the fallback when it wasn't.
func (f Finding) location() string {
	if f.File != "" {
		switch {
		case f.Line > 0:
			return fmt.Sprintf("%s:%d", f.File, f.Line)
		case f.Field != "":
			return fmt.Sprintf("%s (%s)", f.File, f.Field)
		default:
			return f.File
		}
	}
	return f.Subject
}

// String renders one Finding in the three-part standard.
func (f Finding) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", f.location())
	fmt.Fprintf(&b, "  what: %s\n", f.What)
	fmt.Fprintf(&b, "  why:  %s\n", f.Why)
	fmt.Fprintf(&b, "  next: %s\n", f.Next)
	return b.String()
}

// RenderHuman renders every finding, or an explicit clean-pass line when
// there are none — silence on success is not an acceptable report.
func RenderHuman(findings []Finding) string {
	if len(findings) == 0 {
		return "atlas validate: clean — no findings.\n"
	}
	var b strings.Builder
	for i, f := range findings {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.String())
	}
	return b.String()
}

// renderJSONDoc is RenderJSON's shape: the same three-part Findings as data,
// never a thinner shape than the human render.
type renderJSONDoc struct {
	Clean    bool      `json:"clean"`
	Findings []Finding `json:"findings"`
}

// RenderJSON renders findings as the --json mode's payload.
func RenderJSON(findings []Finding) ([]byte, error) {
	if findings == nil {
		findings = []Finding{}
	}
	return json.MarshalIndent(renderJSONDoc{Clean: len(findings) == 0, Findings: findings}, "", "  ")
}

// explain maps a jsonschema ErrorKind to the why/next halves of the
// three-part standard, reusable per error KIND rather than authored fresh
// per occurrence. Covers every keyword the five spec/ schemas actually use;
// falls back to a generic pair for anything else rather than panicking or
// omitting the why/next fields.
func explain(k jsonschema.ErrorKind) (why, next string) {
	switch k.(type) {
	case *kind.Required:
		return "Atlas can't trust an instance that's missing a field its schema declares mandatory — every consumer downstream assumes the field is there.",
			"Add the missing field."
	case *kind.Const:
		return "This field is pinned to one exact value on purpose (e.g. schema_version) so every consumer can rely on it without checking.",
			"Set the field to the exact value the schema requires."
	case *kind.Enum:
		return "This field only accepts one of a fixed, named set of values — anything else can't be resolved by the rest of Atlas.",
			"Use one of the values the schema allows."
	case *kind.Type:
		return "The value's JSON type doesn't match what this field is declared to hold.",
			"Change the value to the expected type."
	case *kind.Pattern:
		return "This field is a name other parts of Atlas parse or reference (e.g. a kebab-case id) — an unexpected shape breaks that.",
			"Reshape the value to match the required pattern."
	case *kind.MinItems:
		return "This list must have at least one entry for the artifact to mean anything.",
			"Add at least one item."
	case *kind.UniqueItems:
		return "A byte-identical duplicate here is dead weight or a sign something was copy-pasted without being changed.",
			"Remove the duplicate entry."
	case *kind.AdditionalProperties:
		return "Atlas schemas are closed on purpose — an unexpected field is usually a typo or a stale field name.",
			"Remove the field, or check it against spec/ for a typo."
	case *kind.AnyOf:
		return "This value doesn't match any of the shapes this field is allowed to take.",
			"Match one of the shapes the schema allows."
	default:
		return "This value doesn't satisfy the schema.",
			"Check the field against the schema in spec/ for the exact rule."
	}
}

// schemaFindings flattens a schema validation failure into Findings. err is
// expected to be a *jsonschema.ValidationError (nil means the instance
// validated cleanly) — anything else is a compile/IO-level error the caller
// handles separately (see schema.go), not a Finding. node is the same
// instance decoded as a yaml.Node tree (nil if that decode failed, or
// wasn't attempted) — used to resolve each error's real source line;
// resolution is best-effort, never fatal to the check itself.
//
// KNOWN LIMITATION: an anyOf failure reports one Finding per failing branch
// (each branch's leaf cause), not one collapsed Finding for the field —
// harmless today (only two fields in the whole spec/ set use anyOf, each
// with two branches) but worth revisiting if a schema grows a wider anyOf.
func schemaFindings(file string, node *yaml.Node, err error) []Finding {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok || ve == nil {
		return nil
	}
	var findings []Finding
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			why, next := explain(e.ErrorKind)
			line := 0
			if n := yamlNodeAt(node, e.InstanceLocation); n != nil {
				line = n.Line
			}
			findings = append(findings, Finding{
				Check: "schema",
				File:  file,
				Line:  line,
				Field: fieldPath(e.InstanceLocation),
				What:  e.ErrorKind.LocalizedString(printer),
				Why:   why,
				Next:  next,
			})
			return
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)
	return findings
}

// fieldPath renders a jsonschema InstanceLocation as a JSON-pointer-ish path.
func fieldPath(loc []string) string {
	if len(loc) == 0 {
		return "(root)"
	}
	return "/" + strings.Join(loc, "/")
}
