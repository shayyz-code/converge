package crdt

import "time"

type LWWRegister[T any] struct {
	Value     T
	Timestamp time.Time
	Node      string
}

func NewLWWRegister[T any](value T, timestamp time.Time, node string) LWWRegister[T] {
	return LWWRegister[T]{Value: value, Timestamp: timestamp, Node: node}
}

func (r *LWWRegister[T]) Set(value T, timestamp time.Time, node string) {
	next := LWWRegister[T]{Value: value, Timestamp: timestamp, Node: node}
	r.Merge(next)
}

func (r *LWWRegister[T]) Merge(other LWWRegister[T]) {
	if other.Timestamp.After(r.Timestamp) {
		*r = other
		return
	}
	if other.Timestamp.Equal(r.Timestamp) && other.Node > r.Node {
		*r = other
	}
}
