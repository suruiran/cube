package logx

import (
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
)

func Func(fn any) slog.Attr {
	fv := reflect.ValueOf(fn)
	fobj := runtime.FuncForPC(fv.Pointer())
	file, line := fobj.FileLine(fobj.Entry())
	return slog.String("Func", fmt.Sprintf("%s %s %d", fobj.Name(), file, line))
}
