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
	rs := runtime.Callers(skip+2, pc)
	var sb strings.Builder
	for i := range rs {
		fc := runtime.FuncForPC(pc[i])
		file, line := fc.FileLine(pc[i])
		sb.WriteString(file)
		sb.WriteByte(':')
		sb.WriteString(strconv.FormatInt(int64(line), 10))

		sb.WriteString(" (")
		sb.WriteString(FuncName(fc))
		sb.WriteString(")\r\n")
	}
	return sb.String()
}
