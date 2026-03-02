package logx

import (
	"log/slog"

	"github.com/suruiran/cube"
)

func Error(e error) slog.Attr {
	return slog.String("error", e.Error())
}

func Recovered(v any) slog.Attr {
	if ev, ok := v.(error); ok {
		return slog.String("panic", ev.Error())
	}
	return slog.Any("panic", v)
}

type StacktraceOptions struct {
	Skip int
	Size int
}

func ErrorWithStacktrace(e error, opts *StacktraceOptions) slog.Attr {
	if opts == nil {
		opts = &StacktraceOptions{
			Skip: 2,
			Size: 20,
		}
	}
	return slog.Group(
		"errtrace",
		slog.String("error", e.Error()),
		slog.String("trace", cube.ReadStack(opts.Size, opts.Skip)),
	)
}

func RecoveredWithStacktrace(pv any, opts *StacktraceOptions) slog.Attr {
	if opts == nil {
		opts = &StacktraceOptions{
			Skip: 2,
			Size: 20,
		}
	}
	var item slog.Attr
	if ev, ok := pv.(error); ok {
		item = slog.String("panic", ev.Error())
	} else {
		item = slog.Any("panic", pv)
	}
	return slog.Group(
		"panictrace",
		item,
		slog.String("trace", cube.ReadStack(opts.Size, opts.Skip)),
	)
}
