package seqs

import (
	"iter"
	"slices"
)

type Op[T any] struct {
	Seq iter.Seq[T]
}

func New[T any](seq iter.Seq[T]) Op[T] {
	return Op[T]{Seq: seq}
}

func FromSlice[T any](sv []T) Op[T] {
	return New(func(yield func(T) bool) {
		for _, v := range sv {
			if !yield(v) {
				return
			}
		}
	})
}

type Pair[T any, U any] struct {
	Key   T
	Value U
}

func FromSeq2[T any, U any](seq iter.Seq2[T, U]) Op[Pair[T, U]] {
	return Op[Pair[T, U]]{Seq: func(yield func(Pair[T, U]) bool) {
		for k, v := range seq {
			if !yield(Pair[T, U]{Key: k, Value: v}) {
				return
			}
		}
	}}
}

func FromSliceWithIndex[T any](sv []T) Op[Pair[int, T]] {
	return FromSeq2(func(yield func(int, T) bool) {
		for i, v := range sv {
			if !yield(i, v) {
				return
			}
		}
	})
}

func (seq Op[T]) MapTo[N any](each func(T) N) Op[N] {
	return Op[N]{
		Seq: func(yield func(N) bool) {
			for ele := range seq.Seq {
				if !yield(each(ele)) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) Map(each func(T) T) Op[T] {
	return Op[T]{
		Seq: func(yield func(T) bool) {
			for ele := range seq.Seq {
				if !yield(each(ele)) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) MapToWithIndex[N any](each func(T, int) N) Op[N] {
	return Op[N]{
		Seq: func(yield func(N) bool) {
			i := -1
			for ele := range seq.Seq {
				i++
				if !yield(each(ele, i)) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) MapWithIndex(each func(T, int) T) Op[T] {
	return Op[T]{
		Seq: func(yield func(T) bool) {
			i := -1
			for ele := range seq.Seq {
				i++
				if !yield(each(ele, i)) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) Filter(op func(T) bool) Op[T] {
	return Op[T]{
		Seq: func(yield func(T) bool) {
			for ele := range seq.Seq {
				if !op(ele) {
					continue
				}
				if !yield(ele) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) FilterWithIndex(op func(T, int) bool) Op[T] {
	return Op[T]{
		Seq: func(yield func(T) bool) {
			i := -1
			for ele := range seq.Seq {
				i++
				if !op(ele, i) {
					continue
				}
				if !yield(ele) {
					return
				}
			}
		},
	}
}

func (seq Op[T]) Reduce[N any](reduce func(N, T) N, init N) N {
	var acc = init
	for ele := range seq.Seq {
		acc = reduce(acc, ele)
	}
	return acc
}

func (seq Op[T]) ReduceWithIndex[N any](reduce func(N, T, int) N, init N) N {
	var acc = init
	i := -1
	for ele := range seq.Seq {
		i++
		acc = reduce(acc, ele, i)
	}
	return acc
}

func (seq Op[T]) Collect() []T {
	return slices.Collect(seq.Seq)
}
