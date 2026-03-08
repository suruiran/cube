package cube

type Numeric interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

func Abs[T Numeric](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
