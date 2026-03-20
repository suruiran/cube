package logx

import "log/slog"

type LazyType func() slog.Value

func (l LazyType) LogValue() slog.Value { return l() }

var (
	_ slog.LogValuer = (LazyType)(nil)
)

func Lazy(key string, f func() slog.Value) slog.Attr { return slog.Any(key, LazyType(f)) }

func LazyGroup(key string, f func() []slog.Attr) slog.Attr {
	return slog.Any(key, LazyType(func() slog.Value { return slog.GroupValue(f()...) }))
}
