package seqs

import (
	"iter"
)

func FromSlice[T any](sv []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range sv {
			if !yield(v) {
				return
			}
		}
	}
}

func FromSliceWithIndex[T any](sv []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, v := range sv {
			if !yield(i, v) {
				return
			}
		}
	}
}

type Op[T any] struct {
	Seq iter.Seq[T]
}

func NewOp[T any](seq iter.Seq[T]) Op[T] {
	return Op[T]{Seq: seq}
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

func (seq Op[T]) Map[N any](each func(T) N) Op[N] {
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

func (seq Op[T]) MapWithIndex[N any](each func(T, int) N) Op[N] {
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
