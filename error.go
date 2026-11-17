package cube

import "fmt"

func AsError(v any) error {
	if v == nil {
		return nil
	}
	if err, ok := v.(error); ok {
		return err
	}
	return fmt.Errorf("%v", v)
}

func Recover(ep *error) {
	if rv := recover(); rv != nil {
		*ep = AsError(rv)
	}
}
