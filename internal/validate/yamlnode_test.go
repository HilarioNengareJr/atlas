package validate

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func decodeNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(src), &node); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return &node
}

func TestYamlNodeAtTopLevelField(t *testing.T) {
	node := decodeNode(t, "a: 1\nb: 2\n")
	got := yamlNodeAt(node, []string{"b"})
	if got == nil {
		t.Fatal("expected to find field b")
	}
	if got.Value != "2" {
		t.Fatalf("got value %q, want \"2\"", got.Value)
	}
	if got.Line != 2 {
		t.Fatalf("got line %d, want 2", got.Line)
	}
}

func TestYamlNodeAtNestedField(t *testing.T) {
	node := decodeNode(t, "a:\n  b:\n    c: deep\n")
	got := yamlNodeAt(node, []string{"a", "b", "c"})
	if got == nil || got.Value != "deep" {
		t.Fatalf("got %+v, want value \"deep\"", got)
	}
	if got.Line != 3 {
		t.Fatalf("got line %d, want 3", got.Line)
	}
}

func TestYamlNodeAtArrayIndex(t *testing.T) {
	node := decodeNode(t, "items:\n  - first\n  - second\n")
	got := yamlNodeAt(node, []string{"items", "1"})
	if got == nil || got.Value != "second" {
		t.Fatalf("got %+v, want value \"second\"", got)
	}
}

func TestYamlNodeAtEmptyPathReturnsRoot(t *testing.T) {
	node := decodeNode(t, "a: 1\n")
	got := yamlNodeAt(node, nil)
	if got == nil || got.Kind != yaml.MappingNode {
		t.Fatalf("got %+v, want the root mapping node", got)
	}
}

func TestYamlNodeAtMissingPathReturnsNil(t *testing.T) {
	node := decodeNode(t, "a: 1\n")
	if got := yamlNodeAt(node, []string{"nonexistent"}); got != nil {
		t.Fatalf("got %+v, want nil for a missing key", got)
	}
	if got := yamlNodeAt(node, []string{"a", "too", "deep"}); got != nil {
		t.Fatalf("got %+v, want nil when the path goes past a scalar", got)
	}
}

func TestYamlNodeAtNilRoot(t *testing.T) {
	if got := yamlNodeAt(nil, []string{"a"}); got != nil {
		t.Fatalf("got %+v, want nil for a nil root (best-effort degrade)", got)
	}
}
