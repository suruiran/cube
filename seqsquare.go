package cube

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/suruiran/cube/cmap"
)

type _Seq[K comparable] struct {
	key    K
	square *SeqSquare[K]

	mutex    sync.Mutex
	locked   bool
	waiters  []*_Waiter
	notinmap bool
}

type _Waiter struct {
	ch    chan struct{}
	alive atomic.Bool
}

func (li *_Seq[K]) dorelease() (*_Waiter, bool) {
	idx := 0
	for _, w := range li.waiters {
		if !w.alive.Load() {
			continue
		}
		li.waiters[idx] = w
		idx++
	}
	li.waiters = li.waiters[:idx]

	if len(li.waiters) < 1 {
		li.locked = false
		return nil, false
	}

	w := li.waiters[0]
	li.waiters = li.waiters[1:]
	return w, true
}

func (li *_Seq[K]) release() {
	li.mutex.Lock()

	w, more := li.dorelease()
	if !more {
		li.mutex.Unlock()

		li.square.onidle(li)
		return
	}
	li.mutex.Unlock()

	w.ch <- struct{}{}
}

type SeqSquare[K comparable] struct {
	items *cmap.Map[K, *_Seq[K]]

	idleslock sync.Mutex
	idles     []*_Seq[K]
	inidles   Set[K]
	tmpidles  []*_Seq[K]

	maxwaiters int
	maxkeys    int64
}

func (sl *SeqSquare[K]) onidle(item *_Seq[K]) {
	sl.idleslock.Lock()
	defer sl.idleslock.Unlock()

	if sl.inidles.Has(item.key) {
		return
	}
	sl.idles = append(sl.idles, item)
	sl.inidles.Add(item.key)
}

func (sl *SeqSquare[K]) tryclean() {
	sl.idleslock.Lock()
	sl.idles, sl.tmpidles = sl.tmpidles, sl.idles
	sl.inidles.Clear()
	sl.idleslock.Unlock()

	for _, item := range sl.tmpidles {
		if item.mutex.TryLock() {
			if item.locked {
				item.mutex.Unlock()
				continue
			}
			sl.items.Delete(item.key)
			item.notinmap = true
			item.mutex.Unlock()
		}
	}
	sl.tmpidles = sl.tmpidles[:0]
}

type SeqSquareOptions struct {
	// MaxKeys maximum number of keys, default 1024. It should be greater than your expected.
	MaxKeys       int64
	MaxWaiters    int
	CleanInterval time.Duration
	BucketCount   int
}

func NewSeqSquare[K comparable](ctx context.Context, opts *SeqSquareOptions) *SeqSquare[K] {
	if opts == nil {
		opts = &SeqSquareOptions{}
	}
	if opts.MaxKeys <= 0 {
		opts.MaxKeys = 1024
	}
	if opts.MaxWaiters <= 0 {
		opts.MaxWaiters = 32
	}
	if opts.CleanInterval <= 0 {
		opts.CleanInterval = time.Minute * 10
	}
	if opts.BucketCount <= 0 {
		opts.BucketCount = 16
	}
	obj := &SeqSquare[K]{
		inidles:    make(Set[K]),
		maxwaiters: opts.MaxWaiters,
		maxkeys:    opts.MaxKeys,
		items:      cmap.New[K, *_Seq[K]](uint64(opts.BucketCount)),
	}
	Fly(func() {
		ticker := time.NewTicker(opts.CleanInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				{
					return
				}
			case <-ticker.C:
				{
					obj.tryclean()
				}
			}
		}
	})
	return obj
}

var (
	ErrSeqSquareQueueFull = errors.New("cube.SeqSquare: queue full")
	ErrSeqSquareKeysFull  = errors.New("cube.SeqSquare: keys full")
)

type IUnlocker interface {
	Unlock()
}

type _ItemPtr[K comparable] struct{ item *_Seq[K] }

func (handle _ItemPtr[K]) Unlock() { handle.item.release() }

var _ IUnlocker = _ItemPtr[int]{}

func (sl *SeqSquare[K]) Acquire(ctx context.Context, key K) (IUnlocker, error) {
	var seq *_Seq[K]
	var err error

	for {
		seq, _, err = sl.items.GetOrCompute(
			key,
			func() (*_Seq[K], error) {
				if sl.maxkeys > 0 && sl.items.ApproxLen() >= int(sl.maxkeys) {
					return nil, ErrSeqSquareKeysFull
				}
				return &_Seq[K]{square: sl, key: key}, nil
			},
		)
		if err != nil {
			return nil, err
		}

		seq.mutex.Lock()
		if seq.notinmap {
			seq.mutex.Unlock()
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			continue
		}
		break
	}

	if !seq.locked {
		seq.locked = true
		seq.mutex.Unlock()
		return _ItemPtr[K]{item: seq}, nil
	}

	if sl.maxwaiters > 0 && len(seq.waiters) >= sl.maxwaiters {
		seq.mutex.Unlock()
		return nil, ErrSeqSquareQueueFull
	}

	var waiter = &_Waiter{ch: make(chan struct{})}
	alive := &waiter.alive
	alive.Store(true)
	ch := waiter.ch

	seq.waiters = append(seq.waiters, waiter)

	seq.mutex.Unlock()

	for {
		select {
		case <-ctx.Done():
			{
				alive.Store(false)
				select {
				case <-ch:
					{
						seq.release()
					}
				default:
				}
				return nil, ctx.Err()
			}
		case <-ch:
			{
				return _ItemPtr[K]{item: seq}, nil
			}
		}
	}
}
