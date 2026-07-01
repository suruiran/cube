package cube

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func Must2[A any, B any](a A, b B, err error) (A, B) {
	if err != nil {
		panic(err)
	}
	return a, b
}

func Must3[A any, B any, C any](a A, b B, c C, err error) (A, B, C) {
	if err != nil {
		panic(err)
	}
	return a, b, c
}
