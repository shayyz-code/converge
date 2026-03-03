package crdt

import (
	"testing"
	"time"
)

func TestLWWRegister(t *testing.T) {
	now := time.Now().UTC()
	r := NewLWWRegister("a", now, "node-a")
	// Earlier update should not override
	r.Set("b", now.Add(-time.Second), "node-b")
	if r.Value != "a" {
		t.Fatalf("expected a, got %v", r.Value)
	}
	// Later update should override
	r.Set("c", now.Add(time.Second), "node-b")
	if r.Value != "c" {
		t.Fatalf("expected c, got %v", r.Value)
	}
	// Equal timestamp resolves by node id (greater wins)
	r.Merge(LWWRegister[string]{Value: "d", Timestamp: r.Timestamp, Node: "z-node"})
	if r.Value != "d" {
		t.Fatalf("expected d by tie-break, got %v", r.Value)
	}
}
