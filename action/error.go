package action

import "fmt"

type IHttpError interface {
	error
	Code() int
}

type _SimpleHttpError struct {
	code int
	fmt  string
	args []any
}

func (she *_SimpleHttpError) Code() int {
	return she.code
}

func (she *_SimpleHttpError) Error() string {
	return fmt.Sprintf(she.fmt, she.args...)
}

var _ IHttpError = (*_SimpleHttpError)(nil)

func NewHttpError(code int, fmt string, args ...any) IHttpError {
	return &_SimpleHttpError{
		code: code,
		fmt:  fmt,
		args: args,
	}
}
