package rbc

import (
	"context"
	"reflect"
	"sync"
	"unsafe"
)

var (
	pools sync.Map
)

type _TypedPool struct {
	*sync.Pool
	OnPut  func(v any) bool
	memclr func(ptr unsafe.Pointer)
}

type _PoolItem struct {
	anyv any
	uptr unsafe.Pointer
}

func mkmemclr(typ reflect.Type) func(ptr unsafe.Pointer) {
	clrfn, ok := memclrs[typ]
	if ok {
		return clrfn
	}
	return func(ptr unsafe.Pointer) {
		on_std_reflect_call(typ, _LogItemMemclr)
		reflect.NewAt(typ, ptr).Elem().SetZero()
	}
}

var (
	onputs = map[reflect.Type]func(v any) bool{}
)

// onput: return true if the item should be put back to the pool
func RegisterOnPut[T any](onput func(v any) bool) {
	onputs[reflect.TypeFor[T]()] = onput
}

func trymkpool(eletype reflect.Type) any {
	poolv, _ := pools.LoadOrStore(eletype, &_TypedPool{
		Pool: &sync.Pool{
			New: func() any {
				cons, ok := constructors[eletype]
				if !ok {
					val := reflect.New(eletype)
					on_std_reflect_call(eletype, _LogItemConstructor)
					return &_PoolItem{
						anyv: val.Interface(),
						uptr: val.UnsafePointer(),
					}
				}
				val, uptr := cons()
				return &_PoolItem{
					anyv: val,
					uptr: uptr,
				}
			},
		},
		OnPut:  onputs[eletype],
		memclr: mkmemclr(eletype),
	})
	return poolv
}

func with_arena_internal(
	ctx context.Context,
	fnc func(ctx context.Context) error,
	log func(item *_PoolItem, pool *_TypedPool),
) error {
	return fnc(
		WithOnDerefNil(
			ctx,
			func(ctx DerefNilContext, eletype reflect.Type) (unsafe.Pointer, error) {
				poolv, ok := pools.Load(eletype)
				if !ok {
					poolv = trymkpool(eletype)
				}
				pool := poolv.(*_TypedPool)
				item := pool.Get().(*_PoolItem)
				log(item, pool)
				return item.uptr, nil
			},
		),
	)
}

func _put(item *_PoolItem, pool *_TypedPool) {
	if pool.OnPut != nil && !pool.OnPut(item.anyv) {
		return
	}
	pool.memclr(item.uptr)
	pool.Put(item)
}

// WithArenaUnsafe
// This is concurrency safe, GC safe, but value unsafe.
func WithArenaUnsafe(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := sync.Map{}
	defer func() {
		arena.Range(func(itemav, poolav any) bool {
			item := itemav.(*_PoolItem)
			pool := poolav.(*_TypedPool)
			_put(item, pool)
			return true
		})
		arena.Clear()
	}()

	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, pool *_TypedPool) { arena.Store(item, pool) },
	)
}

// WithArena
// This is concurrency safe, GC safe, and value safe.
func WithArena(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := sync.Map{}
	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, _pool *_TypedPool) { arena.Store(item, 1) },
	)
}

// WithLocalArenaUnsafe
// This is GC safe, and concurrency unsafe, value unsafe.
func WithLocalArenaUnsafe(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := make(map[*_PoolItem]*_TypedPool, 64)
	defer func() {
		for item := range arena {
			pool := arena[item]
			_put(item, pool)
		}
		clear(arena)
	}()

	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, pool *_TypedPool) { arena[item] = pool },
	)
}

// WithLocalArena
// This is GC safe, value safe and concurrency unsafe.
func WithLocalArena(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := make([]*_PoolItem, 0, 64)
	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, _pool *_TypedPool) {
			arena = append(arena, item)
		},
	)
}
