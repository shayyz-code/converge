package crdt

type GCounter struct {
	counts map[string]uint64
}

func NewGCounter() *GCounter {
	return &GCounter{counts: map[string]uint64{}}
}

func (c *GCounter) Inc(node string, delta uint64) {
	c.counts[node] = c.counts[node] + delta
}

func (c *GCounter) Value() uint64 {
	var sum uint64
	for _, v := range c.counts {
		sum += v
	}
	return sum
}

func (c *GCounter) Merge(other *GCounter) {
	for node, val := range other.counts {
		if val > c.counts[node] {
			c.counts[node] = val
		}
	}
}
