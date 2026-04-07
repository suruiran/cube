package cube

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(v T) {
	s[v] = struct{}{}
}

func (s Set[T]) Has(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) Del(v T) {
	delete(s, v)
}

func (s Set[T]) Len() int {
	return len(s)
}

func (s Set[T]) Clear() Set[T] {
	clear(s)
	return s
}

func (s Set[T]) AsSlice() []T {
	sli := make([]T, 0, len(s))
	for v := range s {
		sli = append(sli, v)
	}
	return sli
}
