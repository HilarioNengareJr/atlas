package validate

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func decodePlanYAML(t *testing.T, src string) any {
	t.Helper()
	var v any
	if err := yaml.Unmarshal([]byte(src), &v); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return v
}

func TestCheckP9NoDuplicates(t *testing.T) {
	doc := decodePlanYAML(t, `
steps:
  - id: s1
    description: a
    done_when: a-done
  - id: s2
    description: b
    done_when: b-done
`)
	if findings := CheckP9("plan.yaml", doc); len(findings) != 0 {
		t.Fatalf("no-duplicate plan = %d findings, want 0: %+v", len(findings), findings)
	}
}

func TestCheckP9SameIDDifferentContent(t *testing.T) {
	doc := decodePlanYAML(t, `
steps:
  - id: s1
    description: first
    done_when: a
  - id: s1
    description: second, different
    done_when: b
`)
	findings := CheckP9("plan.yaml", doc)
	if len(findings) != 1 {
		t.Fatalf("duplicate id, different content = %d findings, want exactly 1: %+v", len(findings), findings)
	}
	if findings[0].Check != "p9" {
		t.Fatalf("finding.Check = %q, want \"p9\"", findings[0].Check)
	}
}

func TestCheckP9SameIDIdenticalContentNotReported(t *testing.T) {
	// Byte-identical duplicate steps are already a schema-check failure
	// (plan.schema.yaml's uniqueItems) — P9 must not double-report them.
	doc := decodePlanYAML(t, `
steps:
  - id: s1
    description: same
    done_when: a
  - id: s1
    description: same
    done_when: a
`)
	if findings := CheckP9("plan.yaml", doc); len(findings) != 0 {
		t.Fatalf("byte-identical duplicate steps = %d findings, want 0 (uniqueItems' job): %+v", len(findings), findings)
	}
}

func TestCheckP9ThreeWaySameIDReportsOnce(t *testing.T) {
	// Three steps sharing one id, pairwise different, must not multiply
	// into more than one Finding for that id.
	doc := decodePlanYAML(t, `
steps:
  - id: s1
    description: first
    done_when: a
  - id: s1
    description: second
    done_when: b
  - id: s1
    description: third
    done_when: c
`)
	findings := CheckP9("plan.yaml", doc)
	if len(findings) != 1 {
		t.Fatalf("three-way same-id plan = %d findings, want exactly 1 (not one per pair): %+v", len(findings), findings)
	}
}

func TestCheckP9IgnoresMalformedDocument(t *testing.T) {
	if findings := CheckP9("x.yaml", "not-an-object"); findings != nil {
		t.Fatalf("malformed document = %+v, want nil (schema check's job to report)", findings)
	}
	if findings := CheckP9("x.yaml", map[string]any{"title": "no steps field"}); findings != nil {
		t.Fatalf("missing steps field = %+v, want nil", findings)
	}
}
