package cube

type UniqueSlice[T any, K comparable] struct {
	V      []T
	mapv   map[K]struct{}
	keyof  func(*T) K
	factor int
}

func NewUniqueSlice[T any, K comparable](keyof func(*T) K) *UniqueSlice[T, K] {
	return &UniqueSlice[T, K]{
		keyof:  keyof,
		factor: 16,
	}
}

func (s *UniqueSlice[T, K]) WithFactor(factor int) *UniqueSlice[T, K] {
	s.factor = factor
	return s
}

func (s *UniqueSlice[T, K]) Reindex() *UniqueSlice[T, K] {
	s.mapv = nil
	return s
}

func (s *UniqueSlice[T, K]) Push(v T) bool {
	key := s.keyof(&v)
	if s.Has(key) {
		return false
	}
	s.V = append(s.V, v)
	if s.mapv != nil {
		s.mapv[key] = struct{}{}
	}
	return true
}

func (s *UniqueSlice[T, K]) Has(key K) bool {
	sl := len(s.V)
	if sl < s.factor {
		for i := range sl {
			if s.keyof(&s.V[i]) == key {
				return true
			}
		}
		return false
	}
	if s.mapv == nil {
		s.mapv = make(map[K]struct{}, sl)
		for i := range sl {
			s.mapv[s.keyof(&s.V[i])] = struct{}{}
		}
	}
	_, ok := s.mapv[key]
	return ok
}
