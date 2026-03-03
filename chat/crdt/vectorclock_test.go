package crdt

import "testing"

func TestVectorClockCompare(t *testing.T) {
	a := NewVectorClock()
	b := NewVectorClock()

	a.Tick("A")
	a.Tick("A")
	b.Tick("A")
	b.Tick("B")

	// a: A=2
	// b: A=1, B=1
	// concurrent
	if cmp := a.Compare(b); cmp != 0 {
		t.Fatalf("expected concurrent, got %d", cmp)
	}

	b.Tick("A")
	// b: A=2, B=1 >= a
	if cmp := a.Compare(b); cmp != -1 {
		t.Fatalf("expected a < b, got %d", cmp)
	}

	a.Tick("B")
	a.Tick("B")
	// a: A=2, B=2 > b
	if cmp := a.Compare(b); cmp != 1 {
		t.Fatalf("expected a > b, got %d", cmp)
	}
}
