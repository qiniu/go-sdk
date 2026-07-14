//go:build unit

package toposort

import (
	"strings"
	"testing"
)

func TestToposort(t *testing.T) {
	sorted, err := Toposort([]Edge{
		{"A", "B"},
		{"A", "C"},
		{"B", "D"},
		{"C", "D"},
		{"E", nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertBefore(t, sorted, "A", "B")
	assertBefore(t, sorted, "A", "C")
	assertBefore(t, sorted, "B", "D")
	assertBefore(t, sorted, "C", "D")
	if !contains(sorted, "E") {
		t.Fatalf("missing unconnected vertex: %v", sorted)
	}
}

func TestToposortDuplicateEdges(t *testing.T) {
	sorted, err := Toposort([]Edge{
		{"A", "B"},
		{"A", "B"},
		{"B", "C"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertBefore(t, sorted, "A", "B")
	assertBefore(t, sorted, "B", "C")
}

func TestToposortCycle(t *testing.T) {
	_, err := Toposort([]Edge{
		{"A", "B"},
		{"B", "C"},
		{"C", "A"},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestToposortSelfEdge(t *testing.T) {
	if _, err := Toposort([]Edge{{"A", "A"}}); err == nil {
		t.Fatal("expected self-edge error")
	}
}

func TestToposortR(t *testing.T) {
	sorted, err := ToposortR([]Edge{{"A", "B"}, {"B", "C"}})
	if err != nil {
		t.Fatal(err)
	}
	assertBefore(t, sorted, "C", "B")
	assertBefore(t, sorted, "B", "A")
}

func assertBefore(t *testing.T, sorted []interface{}, first, second string) {
	t.Helper()
	firstIndex := indexOf(sorted, first)
	secondIndex := indexOf(sorted, second)
	if firstIndex == -1 || secondIndex == -1 || firstIndex >= secondIndex {
		t.Fatalf("expected %q before %q in %v", first, second, sorted)
	}
}

func contains(sorted []interface{}, expected string) bool {
	return indexOf(sorted, expected) != -1
}

func indexOf(sorted []interface{}, expected string) int {
	for index, vertex := range sorted {
		if vertex == expected {
			return index
		}
	}
	return -1
}
