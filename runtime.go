package cube

import (
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

func FuncName(act any) string {
	actv := reflect.ValueOf(act)
	fncname := runtime.FuncForPC(actv.Pointer()).Name()
	return fncname
}

func ReadStack(skip int, size int) string {
	pc := make([]uintptr, size)
	n := runtime.Callers(skip, pc)
	if n == 0 {
		return "empty stack"
	}
	var sb strings.Builder
	frames := runtime.CallersFrames(pc[:n])
	for {
		frame, more := frames.Next()

		sb.WriteString(frame.File)
		sb.WriteByte(':')
		sb.WriteString(strconv.FormatInt(int64(frame.Line), 10))

		sb.WriteString(" (")
		sb.WriteString(frame.Function)
		sb.WriteString(")\r\n")

		if !more {
			break
		}
	}
	return sb.String()
}
