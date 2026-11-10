package cube

import "sync"

type TypedPool[T any] struct {
	pool  sync.Pool
	reset func(T)
}

func NewTypedPoolWithReset[T any](constructor func() T, reset func(val T)) *TypedPool[T] {
	if reset == nil {
		reset = func(val T) {}
	}
	return &TypedPool[T]{
		pool: sync.Pool{
			New: func() any {
				return constructor()
			},
		},
		reset: reset,
	}
}

func NewTypedPool[T any](constructor func() T) *TypedPool[T] {
	return NewTypedPoolWithReset(constructor, nil)
}

func (tp *TypedPool[T]) Get() T {
	return tp.pool.Get().(T)
}

func (tp *TypedPool[T]) Put(v T) {
	tp.reset(v)
	tp.pool.Put(v)
}
