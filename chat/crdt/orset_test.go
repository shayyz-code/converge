package crdt

import "testing"

func TestORSetAddRemoveMerge(t *testing.T) {
	a := NewORSet[string]("A")
	b := NewORSet[string]("B")

	a.Add("x")
	b.Add("y")
	b.Add("x")

	// Merge both ways should converge
	a.Merge(b)
	b.Merge(a)

	if !a.Has("x") || !a.Has("y") {
		t.Fatalf("expected x and y in set after merge, got %v", a.Values())
	}
	// Remove x on A
	a.Remove("x")

	// Merge again
	b.Merge(a)
	a.Merge(b)

	if a.Has("x") {
		t.Fatalf("expected x to be removed after OR-Remove")
	}
	if !a.Has("y") {
		t.Fatalf("expected y to remain")
	}
}
