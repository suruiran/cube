package rbc

import (
	"fmt"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
)

type _LogItem struct {
	ptrcast     atomic.Uint64
	deref       atomic.Uint64
	set         atomic.Uint64
	constructor atomic.Uint64
	memclr      atomic.Uint64
}

var (
	TraceStdReflectCall = false
	stdReflectLogs      = sync.Map{}
)

const (
	_LogItemPtrcast = iota
	_LogItemDeref
	_LogItemSet
	_LogItemConstructor
	_LogItemMemclr
)

func on_std_reflect_call(typ reflect.Type, cause int) {
	if !TraceStdReflectCall {
		return
	}
	itemv, ok := stdReflectLogs.Load(typ)
	if !ok {
		itemv, _ = stdReflectLogs.LoadOrStore(typ, &_LogItem{})
	}
	logitem := itemv.(*_LogItem)
	switch cause {
	case _LogItemPtrcast:
		{
			logitem.ptrcast.Add(1)
		}
	case _LogItemDeref:
		{
			logitem.deref.Add(1)
		}
	case _LogItemSet:
		{
			logitem.set.Add(1)
		}
	case _LogItemConstructor:
		{
			logitem.constructor.Add(1)
		}
	case _LogItemMemclr:
		{
			logitem.memclr.Add(1)
		}
	}
}

func PrintStdReflectCallLogs(w io.Writer) {
	if !TraceStdReflectCall {
		return
	}
	stdReflectLogs.Range(func(key, itemv any) bool {
		typ := key.(reflect.Type)
		logitem := itemv.(*_LogItem)
		_, _ = fmt.Fprintf(
			w,
			`{"Type":"%s", "Ptrcast":%d, "Deref":%d, "Set":%d, "New":%d, "Memclr":%d}\n`,
			typ.String(),
			logitem.ptrcast.Load(),
			logitem.deref.Load(),
			logitem.set.Load(),
			logitem.constructor.Load(),
			logitem.memclr.Load(),
		)
		return true
	})
}
