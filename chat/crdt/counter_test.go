package crdt

import "testing"

func TestGCounterAndPNCounter(t *testing.T) {
	g1 := NewGCounter()
	g2 := NewGCounter()

	g1.Inc("A", 2)
	g2.Inc("B", 3)
	g1.Merge(g2)
	g2.Merge(g1)

	if g1.Value() != 5 || g2.Value() != 5 {
		t.Fatalf("expected 5, got %d and %d", g1.Value(), g2.Value())
	}

	pn1 := NewPNCounter()
	pn2 := NewPNCounter()

	pn1.Inc("A", 5)
	pn2.Dec("B", 2)
	pn1.Merge(pn2)
	if pn1.Value() != 3 {
		t.Fatalf("expected 3, got %d", pn1.Value())
	}
}
