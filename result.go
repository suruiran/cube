package cube

type Result[T any] struct {
	Err   error
	Value T
}

func (r *Result[T]) IsErr() bool {
	return r.Err != nil
}

func (r *Result[T]) IsOk() bool {
	return r.Err == nil
}

func (r *Result[T]) Unwrap() T {
	if r.Err == nil {
		return r.Value
	}
	panic(r.Err)
}

func (r *Result[T]) Map[A any](onok func(T) A, onerr func(error) A) A {
	if r.Err == nil {
		return onok(r.Value)
	}
	return onerr(r.Err)
}
