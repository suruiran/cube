package cmap

import (
	"hash/maphash"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

type Bucket[K comparable, V any] struct {
	sync.RWMutex
	Map map[K]V

	_ cpu.CacheLinePad
}

type Map[K comparable, V any] struct {
	Buckets []Bucket[K, V]
	mask    uint64
	seed    maphash.Seed
}

func New[K comparable, V any](size uint64) *Map[K, V] {
	if size < 4 || size&(size-1) != 0 {
		panic("size must be power of 2 and at least 4")
	}

	obj := &Map[K, V]{
		Buckets: make([]Bucket[K, V], size),
		mask:    size - 1,
		seed:    maphash.MakeSeed(),
	}
	for i := range obj.Buckets {
		obj.Buckets[i].Map = make(map[K]V)
	}
	return obj
}

func (cm *Map[K, V]) Entry(key K) Entry[K, V] {
	skey := maphash.Comparable(cm.seed, key)
	return Entry[K, V]{
		Key:    key,
		Bucket: &cm.Buckets[skey&cm.mask],
	}
}

func (cm *Map[K, V]) ApproxLen() int {
	size := len(cm.Buckets)
	lv := int64(0)
	if size > 256 {
		var wg sync.WaitGroup
		groupsize := size / 4
		wg.Add(4)

		for gi := range 4 {
			go func() {
				defer wg.Done()

				begin := gi * groupsize
				end := (gi + 1) * groupsize

				gc := 0
				for i := begin; i < end; i++ {
					bucket := &cm.Buckets[i]
					bucket.RLock()
					gc += len(bucket.Map)
					bucket.RUnlock()
				}

				atomic.AddInt64(&lv, int64(gc))
			}()
		}
		wg.Wait()
	} else {
		for i := range size {
			bucket := &cm.Buckets[i]
			bucket.RLock()
			lv += int64(len(bucket.Map))
			bucket.RUnlock()
		}
	}
	return int(lv)
}

type Entry[K comparable, V any] struct {
	Key    K
	Bucket *Bucket[K, V]
}

func (e Entry[K, V]) Get() (V, bool) {
	bucket := e.Bucket

	bucket.RLock()
	defer bucket.RUnlock()
	v, ok := bucket.Map[e.Key]
	return v, ok
}

func (m Map[K, V]) Get(key K) (V, bool) {
	return m.Entry(key).Get()
}

func (m Map[K, V]) Contains(key K) bool {
	_, ok := m.Get(key)
	return ok
}

func (e Entry[K, V]) GetOrCompute(constructor func() (V, error)) (V, bool, error) {
	bucket := e.Bucket
	key := e.Key

	bucket.RLock()
	v, exists := bucket.Map[key]
	if exists {
		bucket.RUnlock()
		return v, true, nil
	}
	bucket.RUnlock()

	val, err := constructor()
	if err != nil {
		return val, false, err
	}

	bucket.Lock()
	defer bucket.Unlock()

	v, exists = bucket.Map[key]
	if exists {
		return v, true, nil
	}
	bucket.Map[key] = val
	return val, false, nil
}

func (m *Map[K, V]) GetOrCompute(key K, constructor func() (V, error)) (V, bool, error) {
	return m.Entry(key).GetOrCompute(constructor)
}

func (e Entry[K, V]) Set(value V) {
	bucket := e.Bucket

	bucket.Lock()
	defer bucket.Unlock()

	bucket.Map[e.Key] = value
}

func (m *Map[K, V]) Set(key K, value V) {
	m.Entry(key).Set(value)
}

func (e Entry[K, V]) Delete() {
	bucket := e.Bucket

	bucket.Lock()
	defer bucket.Unlock()

	delete(bucket.Map, e.Key)
}

func (m *Map[K, V]) Delete(key K) {
	m.Entry(key).Delete()
}
