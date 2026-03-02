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

//go:linkname _memclr runtime.memclrNoHeapPointers
func _memclr(ptr unsafe.Pointer, n uintptr)

func memclr(ptr unsafe.Pointer, size uintptr) {
	_memclr(ptr, size)
	// if size == 0 {
	// 	return
	// }
	// sh := unsafe.Slice((*byte)(ptr), size)
	// for i := range sh {
	// 	sh[i] = 0
	// }
}

type _TypedPool struct {
	*sync.Pool
	Size  uintptr
	OnPut func(v any) bool
}

type _PoolItem struct {
	anyv any
	uptr unsafe.Pointer
}

var (
	onputs = map[reflect.Type]func(v any) bool{}
)

func RegisterOnPut[T any](onput func(v any) bool) {
	onputs[reflect.TypeFor[T]()] = onput
}

//go:noinline
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
		Size:  eletype.Size(),
		OnPut: onputs[eletype],
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
	if pool.OnPut == nil {
		memclr(item.uptr, pool.Size)
	} else {
		if !pool.OnPut(item.anyv) {
			return
		}
	}
	pool.Put(item)
}

// WithArena
// This is concurrency safe, GC safe, but value unsafe.
func WithArena(ctx context.Context, fnc func(ctx context.Context) error) error {
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

// WithTempArena
// This is concurrency safe, GC safe, and value safe.
func WithTempArena(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := sync.Map{}
	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, pool *_TypedPool) { arena.Store(item, pool) },
	)
}

// WithLocalArena
// This is GC safe, and concurrency unsafe, value unsafe.
func WithLocalArena(ctx context.Context, fnc func(ctx context.Context) error) error {
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

// WithTempLocalArena
// This is GC safe, value safe and concurrency unsafe.
func WithTempLocalArena(ctx context.Context, fnc func(ctx context.Context) error) error {
	arena := make(map[*_PoolItem]*_TypedPool, 64)
	return with_arena_internal(
		ctx,
		fnc,
		func(item *_PoolItem, pool *_TypedPool) { arena[item] = pool },
	)
}
