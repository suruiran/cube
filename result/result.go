package result

import "errors"

type Result[T any] struct {
	err   error
	value T
}

func (r *Result[T]) IsErr() bool {
	return r.err != nil
}

func (r *Result[T]) Err() error {
	return r.err
}

func (r *Result[T]) IsOk() bool {
	return r.err == nil
}

func (r *Result[T]) Must() T {
	if r.err == nil {
		return r.value
	}
	panic(r.err)
}

func (r *Result[T]) OrElse(fallback func(error) T) T {
	if r.err == nil {
		return r.value
	}
	return fallback(r.err)
}

func (r *Result[T]) Map[A any](onok func(T) A, onerr func(error) A) A {
	if r.err == nil {
		return onok(r.value)
	}
	return onerr(r.err)
}

func Ok[T any](val T) Result[T] {
	return Result[T]{
		value: val,
	}
}

var (
	errNil = errors.New("cube.result: nil")
)

func Err[T any](e error) Result[T] {
	if e == nil {
		return Result[T]{
			err: errNil,
		}
	}
	return Result[T]{
		err: e,
	}
}

func From[T any](val T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(val)
}
