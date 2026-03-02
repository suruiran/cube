package cube

import "golang.org/x/exp/constraints"

type Numeric interface {
	constraints.Signed | constraints.Float
}

func Abs[T Numeric](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
