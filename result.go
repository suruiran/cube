package cube

type Result[T any] struct {
	Err   error
	Value T
}
