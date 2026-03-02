package logx

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type MultSender struct {
	Handlers []slog.Handler

	rw    sync.RWMutex
	cache map[slog.Level][]slog.Handler
}

func (ms *MultSender) enabled(ctx context.Context, lvl slog.Level) []slog.Handler {
	ms.rw.RLock()

	handlers, ok := ms.cache[lvl]
	if ok {
		ms.rw.RUnlock()
		return handlers
	}
	ms.rw.RUnlock()

	ms.rw.Lock()
	defer ms.rw.Unlock()

	if handlers, ok = ms.cache[lvl]; ok {
		return handlers
	}

	hs := make([]slog.Handler, 0, len(ms.Handlers))
	for _, h := range ms.Handlers {
		if h.Enabled(ctx, lvl) {
			hs = append(hs, h)
		}
	}
	ms.cache[lvl] = hs
	return hs
}

func (ms *MultSender) Enabled(ctx context.Context, lvl slog.Level) bool {
	return len(ms.enabled(ctx, lvl)) > 0
}

func (ms *MultSender) Handle(ctx context.Context, r slog.Record) error {
	var err error
	for _, h := range ms.enabled(ctx, r.Level) {
		if tmperr := h.Handle(ctx, r); tmperr != nil {
			if err == nil {
				err = tmperr
			} else {
				err = errors.Join(err, tmperr)
			}
		}
	}
	return err
}

func (ms *MultSender) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return ms
	}

	ms.rw.RLock()
	defer ms.rw.RUnlock()

	tmp := MultSender{
		Handlers: make([]slog.Handler, 0, len(ms.Handlers)),
		cache:    map[slog.Level][]slog.Handler{},
	}
	for _, v := range ms.Handlers {
		tmp.Handlers = append(tmp.Handlers, v.WithAttrs(attrs))
	}
	return &tmp
}

func (ms *MultSender) WithGroup(name string) slog.Handler {
	ms.rw.RLock()
	defer ms.rw.RUnlock()

	tmp := MultSender{
		Handlers: make([]slog.Handler, 0, len(ms.Handlers)),
		cache:    map[slog.Level][]slog.Handler{},
	}
	for _, v := range ms.Handlers {
		tmp.Handlers = append(tmp.Handlers, v.WithGroup(name))
	}
	return &tmp
}

func (ms *MultSender) Append(handlers ...slog.Handler) {
	ms.rw.Lock()
	defer ms.rw.Unlock()

	ms.Handlers = append(ms.Handlers, handlers...)
	ms.cache = map[slog.Level][]slog.Handler{}
}

func (ms *MultSender) Remove(idx int) {
	ms.rw.Lock()
	defer ms.rw.Unlock()

	if idx < 0 || idx >= len(ms.Handlers) {
		return
	}
	ms.Handlers = append(ms.Handlers[:idx], ms.Handlers[idx+1:]...)
	ms.cache = map[slog.Level][]slog.Handler{}
}

var _ slog.Handler = (*MultSender)(nil)

func NewMultSender(handlers ...slog.Handler) *MultSender {
	return &MultSender{
		Handlers: handlers,
		cache:    map[slog.Level][]slog.Handler{},
	}
}
