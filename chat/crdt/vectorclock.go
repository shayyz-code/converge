package crdt

type VectorClock struct {
	clock map[string]uint64
}

func NewVectorClock() *VectorClock {
	return &VectorClock{clock: map[string]uint64{}}
}

func (v *VectorClock) Tick(node string) {
	v.clock[node] = v.clock[node] + 1
}

func (v *VectorClock) Merge(other *VectorClock) {
	for node, val := range other.clock {
		if val > v.clock[node] {
			v.clock[node] = val
		}
	}
}

// Compare returns:
// -1 if v < other, 0 if concurrent, 1 if v > other
func (v *VectorClock) Compare(other *VectorClock) int {
	lt, gt := false, false
	seen := map[string]struct{}{}
	for node, val := range v.clock {
		oval := other.clock[node]
		if val < oval {
			lt = true
		} else if val > oval {
			gt = true
		}
		seen[node] = struct{}{}
	}
	for node, oval := range other.clock {
		if _, ok := seen[node]; ok {
			continue
		}
		val := v.clock[node]
		if val < oval {
			lt = true
		} else if val > oval {
			gt = true
		}
	}
	if lt && !gt {
		return -1
	}
	if gt && !lt {
		return 1
	}
	return 0
}
