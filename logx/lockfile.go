package logx

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/gofrs/flock"
)

type LockFile struct {
	mu     sync.Mutex
	file   *os.File
	lock   *flock.Flock
	closed atomic.Bool
}

func OpenLockFile(fp string, flag int, perm os.FileMode) (*LockFile, error) {
	if flag&os.O_APPEND == 0 {
		return nil, fmt.Errorf("file flag must contains os.O_APPEND, %s, %d", fp, flag)
	}

	fv, err := os.OpenFile(fp, flag, perm)
	if err != nil {
		return nil, err
	}
	lock := flock.New(fmt.Sprintf("%s.lock", fp))

	return &LockFile{
		file: fv,
		lock: lock,
	}, nil
}

func (f *LockFile) Write(p []byte) (int, error) {
	if f.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	if err := f.lock.Lock(); err != nil {
		return 0, err
	}
	defer f.lock.Unlock() //nolint:errcheck

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.file.Write(p)
}

func (f *LockFile) Sync() error {
	if f.closed.Load() {
		return io.ErrClosedPipe
	}

	if err := f.lock.Lock(); err != nil {
		return err
	}
	defer f.lock.Unlock() //nolint:errcheck

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.file.Sync()
}

func (f *LockFile) Close() error {
	if !f.closed.CompareAndSwap(false, true) {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.file.Close()
}

func (f *LockFile) Stat() (os.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Stat()
}
