package logx

import (
	"io"
	"sync"
)

type Syncer interface {
	Sync() error
}

type NoBufferedWriter struct {
	lock sync.Mutex
	w    io.Writer
	f    Flusher
	s    Syncer
}

func (nbw *NoBufferedWriter) Write(p []byte) (int, error) {
	nbw.lock.Lock()
	defer nbw.lock.Unlock()

	n, err := nbw.w.Write(p)
	if err != nil {
		return n, err
	}
	if nbw.s != nil {
		return n, nbw.s.Sync()
	}
	if nbw.f != nil {
		return n, nbw.f.Flush()
	}
	return n, nil
}

var _ io.Writer = (*NoBufferedWriter)(nil)

func NewNoBufferedWriter(w io.Writer) *NoBufferedWriter {
	nbw := &NoBufferedWriter{w: w}
	if f, ok := w.(Flusher); ok {
		nbw.f = f
	}
	if s, ok := w.(Syncer); ok {
		nbw.s = s
	}
	return nbw
}
