package crdt

type ORSet[T comparable] struct {
	nodeID  string
	counter uint64
	adds    map[T]map[Dot]struct{}
	removes map[T]map[Dot]struct{}
}

func NewORSet[T comparable](nodeID string) *ORSet[T] {
	return &ORSet[T]{
		nodeID:  nodeID,
		adds:    make(map[T]map[Dot]struct{}),
		removes: make(map[T]map[Dot]struct{}),
	}
}

func (s *ORSet[T]) Add(value T) Dot {
	s.counter++
	dot := Dot{Node: s.nodeID, Counter: s.counter}
	if s.adds[value] == nil {
		s.adds[value] = make(map[Dot]struct{})
	}
	s.adds[value][dot] = struct{}{}
	return dot
}

func (s *ORSet[T]) Remove(value T) {
	dots := s.adds[value]
	if len(dots) == 0 {
		return
	}
	if s.removes[value] == nil {
		s.removes[value] = make(map[Dot]struct{})
	}
	for dot := range dots {
		s.removes[value][dot] = struct{}{}
	}
}

func (s *ORSet[T]) Has(value T) bool {
	adds := s.adds[value]
	if len(adds) == 0 {
		return false
	}
	removes := s.removes[value]
	for dot := range adds {
		if _, removed := removes[dot]; !removed {
			return true
		}
	}
	return false
}

func (s *ORSet[T]) Values() []T {
	out := []T{}
	for value := range s.adds {
		if s.Has(value) {
			out = append(out, value)
		}
	}
	return out
}

func (s *ORSet[T]) Merge(other *ORSet[T]) {
	for value, dots := range other.adds {
		if s.adds[value] == nil {
			s.adds[value] = make(map[Dot]struct{})
		}
		for dot := range dots {
			s.adds[value][dot] = struct{}{}
		}
	}
	for value, dots := range other.removes {
		if s.removes[value] == nil {
			s.removes[value] = make(map[Dot]struct{})
		}
		for dot := range dots {
			s.removes[value][dot] = struct{}{}
		}
	}
	if other.counter > s.counter {
		s.counter = other.counter
	}
}
