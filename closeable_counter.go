package cube

import (
	"errors"
	"sync"
)

type CloseableCounter struct {
	rw      sync.RWMutex
	closed  bool
	counter int64

	waitfor  int64
	waitchan chan struct{}
}

func (cc *CloseableCounter) on_change() {
	if cc.waitchan == nil {
		return
	}
	if cc.waitfor != cc.counter {
		return
	}
	ch := cc.waitchan
	cc.waitchan = nil
	ch <- struct{}{}
}

func (cc *CloseableCounter) Acquire() bool {
	cc.rw.Lock()
	defer cc.rw.Unlock()

	if cc.closed {
		return false
	}
	cc.counter++
	cc.on_change()
	return true
}

func (cc *CloseableCounter) Release() {
	cc.rw.Lock()
	defer cc.rw.Unlock()

	cc.counter--
	cc.on_change()
}

func (cc *CloseableCounter) Close() bool {
	cc.rw.Lock()
	defer cc.rw.Unlock()

	if cc.closed {
		return false
	}
	cc.closed = true
	return true
}

func (cc *CloseableCounter) Closed() bool {
	cc.rw.RLock()
	defer cc.rw.RUnlock()
	return cc.closed
}

func (cc *CloseableCounter) Count() int64 {
	cc.rw.RLock()
	defer cc.rw.RUnlock()
	return cc.counter
}

func (cc *CloseableCounter) Read(fn func(closed bool, count int64) any) any {
	cc.rw.RLock()
	defer cc.rw.RUnlock()
	return fn(cc.closed, cc.counter)
}

func (cc *CloseableCounter) Wait(n int64) {
	cc.rw.Lock()

	if cc.counter == n {
		cc.rw.Unlock()
		return
	}

	if cc.waitchan != nil {
		cc.rw.Unlock()
		panic(errors.New("cube.CloseableCounter: already in waiting"))
	}
	cc.waitchan = make(chan struct{}, 1)
	cc.waitfor = n
	cc.rw.Unlock()

	<-cc.waitchan
}
