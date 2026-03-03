package crdt

type PNCounter struct {
	pos *GCounter
	neg *GCounter
}

func NewPNCounter() *PNCounter {
	return &PNCounter{
		pos: NewGCounter(),
		neg: NewGCounter(),
	}
}

func (c *PNCounter) Inc(node string, delta uint64) {
	c.pos.Inc(node, delta)
}

func (c *PNCounter) Dec(node string, delta uint64) {
	c.neg.Inc(node, delta)
}

func (c *PNCounter) Value() int64 {
	return int64(c.pos.Value()) - int64(c.neg.Value())
}

func (c *PNCounter) Merge(other *PNCounter) {
	c.pos.Merge(other.pos)
	c.neg.Merge(other.neg)
}
