package seqs

import (
	"iter"

	"github.com/suruiran/cube/logic"
)

type Kind int

const (
	Ok Kind = iota
	Skip
	Stop
)

func Pipe[T any, V any](
	input iter.Seq[T],
	ops ...func(ele any) (any, Kind),
) iter.Seq[V] {
	return func(yield func(V) bool) {
	loop:
		for ele := range input {
			var av any = ele
			for _, op := range ops {
				var kind Kind
				av, kind = op(av)
				if kind == Stop {
					return
				}
				if kind == Skip {
					continue loop
				}
			}
			if !yield(av.(V)) {
				return
			}
		}
	}
}

func Op[I any, O any](op func(ele I) (O, Kind)) func(any) (any, Kind) {
	return func(ele any) (any, Kind) {
		ov, kind := op(any(ele).(I))
		return any(ov), kind
	}
}

func Filter[T any](op func(ele T) bool) func(any) (any, Kind) {
	return Op(func(ele T) (T, Kind) {
		if op(ele) {
			return ele, Ok
		}
		return ele, Skip
	})
}

func FilterByValue[T any]() func(any) (any, Kind) {
	return Op(func(ele T) (T, Kind) {
		if logic.All(ele) {
			return ele, Ok
		}
		return ele, Skip
	})
}

func PipePair[K any, V any, OK any, OV any](
	input iter.Seq2[K, V],
	ops ...func(k any, v any) (any, any, Kind),
) iter.Seq2[OK, OV] {
	return func(yield func(OK, OV) bool) {
	loop:
		for k, v := range input {
			var ak any = k
			var av any = v
			for _, op := range ops {
				var kind Kind
				ak, av, kind = op(ak, av)
				if kind == Stop {
					return
				}
				if kind == Skip {
					continue loop
				}
			}
			if !yield(ak.(OK), av.(OV)) {
				return
			}
		}
	}
}

func OpPair[IK any, IV any, OK any, OV any](op func(k IK, v IV) (OK, OV, Kind)) func(any, any) (any, any, Kind) {
	return func(k any, v any) (any, any, Kind) {
		oK, oV, kind := op(any(k).(IK), any(v).(IV))
		return any(oK), any(oV), kind
	}
}

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

func FromMap[T comparable, U any](mv map[T]U) iter.Seq2[T, U] {
	return func(yield func(T, U) bool) {
		for k, v := range mv {
			if !yield(k, v) {
				return
			}
		}
	}
}

func Reverse[T any](sv []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := len(sv) - 1; i >= 0; i-- {
			if !yield(sv[i]) {
				return
			}
		}
	}
}

func ReverseWithIndex[T any](sv []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i := len(sv) - 1; i >= 0; i-- {
			if !yield(i, sv[i]) {
				return
			}
		}
	}
}
