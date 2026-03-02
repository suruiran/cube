package logx

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/suruiran/cube"
)

type Flusher interface {
	Flush() error
}

type AutoSaveWriter struct {
	w io.Writer
	c io.Closer
	f Flusher

	closed atomic.Bool
}

func (afw *AutoSaveWriter) Close() error {
	if !afw.closed.CompareAndSwap(false, true) {
		return nil
	}

	var err error
	if afw.f != nil {
		err = afw.f.Flush()
	}
	if afw.c != nil {
		tmp := afw.c.Close()
		if tmp != nil {
			if err == nil {
				err = tmp
			} else {
				err = errors.Join(err, tmp)
			}
		}
	}
	return err
}

var _ io.WriteCloser = (*AutoSaveWriter)(nil)

func (afw *AutoSaveWriter) Write(p []byte) (n int, err error) {
	if afw.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	return afw.w.Write(p)
}

func NewAutoSaveWriter(w io.Writer) *AutoSaveWriter {
	afw := &AutoSaveWriter{w: w}
	cobj, ok := w.(io.Closer)
	if ok {
		afw.c = cobj
	}
	fobj, ok := w.(Flusher)
	if ok {
		afw.f = fobj
	}
	cube.OnDeath(func(wg *sync.WaitGroup) {
		wg.Add(1)
		defer wg.Done()
		_ = afw.Close()
	})
	return afw
}
