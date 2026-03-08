package cube

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type _LockItem struct {
	busy   atomic.Bool
	key    any
	onidle func(*_LockItem)

	mutex   sync.Mutex
	locked  bool
	waiters []*_Waiter
}

type _Waiter struct {
	ch    chan struct{}
	alive atomic.Bool
}

func (li *_LockItem) dorelease() (*_Waiter, bool) {
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
		li.busy.Store(false)
		return nil, false
	}

	w := li.waiters[0]
	li.waiters = li.waiters[1:]
	li.busy.Store(true)
	return w, true
}

func (li *_LockItem) release() {
	li.mutex.Lock()

	w, more := li.dorelease()
	if !more {
		li.mutex.Unlock()
		li.onidle(li)
		return
	}
	li.mutex.Unlock()
	w.ch <- struct{}{}
}

type SeqSquare[K comparable] struct {
	items       sync.Map
	inacc_count atomic.Int64

	idleslock sync.Mutex
	idles     []*_LockItem
	inidles   Set[K]
	tmpidles  []*_LockItem

	maxwaiters int
	maxkeys    int64
}

func (sl *SeqSquare[K]) onidle(item *_LockItem) {
	sl.idleslock.Lock()
	defer sl.idleslock.Unlock()

	_key := item.key.(K)
	if sl.inidles.Has(_key) {
		return
	}
	sl.idles = append(sl.idles, item)
	sl.inidles.Add(_key)
}

func (sl *SeqSquare[K]) try_recount_keys(overlap_factor float64) {
	begin := sl.inacc_count.Load()
	rc := 0
	sl.items.Range(func(_, _ any) bool {
		rc++
		return true
	})
	inc := max(int64(float64((sl.inacc_count.Load()-begin))*overlap_factor), 0)
	sl.inacc_count.Store(int64(rc) + inc)
}

func (sl *SeqSquare[K]) tryclean() {
	sl.idleslock.Lock()
	sl.idles, sl.tmpidles = sl.tmpidles, sl.idles
	sl.inidles.Clear()
	sl.idleslock.Unlock()

	for _, item := range sl.tmpidles {
		if item.locked || item.busy.Load() {
			continue
		}
		if _, ok := sl.items.LoadAndDelete(item.key); ok {
			sl.inacc_count.Add(-1)
		}
	}
	sl.tmpidles = sl.tmpidles[:0]
}

type SeqSquareOptions struct {
	MaxKeys                      int64
	MaxKeysToleranceMargin       float64
	RecountOverlapCorrectionRate float64
	// MaxWaiters maximum number of waiters per key
	MaxWaiters int
	// CleanInterval how often to clean up idle items
	CleanInterval time.Duration
	// RecountKeysSteps how many intervals to recount keys
	RecountKeysSteps int
}

func NewSeqSquare[K comparable](ctx context.Context, opts *SeqSquareOptions) *SeqSquare[K] {
	if opts == nil {
		opts = &SeqSquareOptions{}
	}
	if opts.MaxKeysToleranceMargin <= 1 {
		opts.MaxKeysToleranceMargin = 1.2
	}
	if opts.MaxKeys <= 0 {
		opts.MaxKeys = 256
	}
	if opts.MaxWaiters <= 0 {
		opts.MaxWaiters = 32
	}
	if opts.CleanInterval <= 0 {
		opts.CleanInterval = time.Minute * 10
	}
	if opts.RecountKeysSteps <= 0 {
		opts.RecountKeysSteps = 6
	}
	if opts.RecountOverlapCorrectionRate <= 0 {
		opts.RecountOverlapCorrectionRate = 0.6
	}

	obj := &SeqSquare[K]{
		inidles:    make(Set[K]),
		maxwaiters: int(float64(opts.MaxWaiters) * opts.MaxKeysToleranceMargin),
		maxkeys:    opts.MaxKeys,
	}
	Fly(func() {
		ticker := time.NewTicker(opts.CleanInterval)
		defer ticker.Stop()

		lc := 1
		for {
			select {
			case <-ctx.Done():
				{
					return
				}
			case <-ticker.C:
				{
					obj.tryclean()
					if lc%opts.RecountKeysSteps == 0 {
						obj.try_recount_keys(opts.RecountOverlapCorrectionRate)
					}
					lc++
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

var (
	_lockitems_pool = sync.Pool{
		New: func() any { return &_LockItem{} },
	}
)

type IUnlocker interface {
	Unlock()
}

type _ItemPtr struct{ item *_LockItem }

func (handle _ItemPtr) Unlock() { handle.item.release() }

var _ IUnlocker = _ItemPtr{}

func (sl *SeqSquare[K]) Acquire(ctx context.Context, key K) (IUnlocker, error) {
	var keyav any = key
	kc := sl.inacc_count.Add(1)
	if sl.maxkeys > 0 && kc > sl.maxkeys {
		sl.inacc_count.Add(-1)
		return nil, ErrSeqSquareKeysFull
	}

	_item_av := _lockitems_pool.Get()
	itemav, loaded := sl.items.LoadOrStore(keyav, _item_av)
	if loaded {
		_lockitems_pool.Put(_item_av)
		sl.inacc_count.Add(-1)
	}
	item := itemav.(*_LockItem)

	item.busy.Store(true)
	item.mutex.Lock()

	if item.onidle == nil {
		item.onidle = sl.onidle
		item.key = keyav
	}

	if !item.locked {
		item.locked = true
		item.mutex.Unlock()
		return _ItemPtr{item: item}, nil
	}

	if sl.maxwaiters > 0 && len(item.waiters) >= sl.maxwaiters {
		item.mutex.Unlock()
		return nil, ErrSeqSquareQueueFull
	}

	var waiter = &_Waiter{
		ch: make(chan struct{}, 1),
	}
	alive := &waiter.alive
	alive.Store(true)
	ch := waiter.ch

	item.waiters = append(item.waiters, waiter)

	item.mutex.Unlock()

	for {
		select {
		case <-ctx.Done():
			{
				alive.Store(false)
				select {
				case <-ch:
					{
						item.release()
					}
				default:
				}
				return nil, ctx.Err()
			}
		case <-ch:
			{
				return _ItemPtr{item: item}, nil
			}
		}
	}
}
