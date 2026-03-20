package cube

import (
	"iter"
)

type _VecMapPair[K comparable, V any] struct {
	key   K
	value V
	ok    bool
}

type VecMap[K comparable, V any] struct {
	vec     []_VecMapPair[K, V]
	vecsize int
	mapv    map[K]V
	factor  int
}

var (
	DefaultVecMapFactor = 12
)

func NewVecMap[K comparable, V any]() *VecMap[K, V] {
	factor := DefaultVecMapFactor
	return &VecMap[K, V]{
		factor: factor,
		vec:    make([]_VecMapPair[K, V], 0, factor),
	}
}

func (m *VecMap[K, V]) Len() int {
	if m.mapv != nil {
		return len(m.mapv)
	}
	return m.vecsize
}

func (m *VecMap[K, V]) Get(key K) (V, bool) {
	if m.mapv != nil {
		v, ok := m.mapv[key]
		return v, ok
	}
	for i := range len(m.vec) {
		item := &m.vec[i]
		if item.ok && item.key == key {
			return item.value, true
		}
	}
	var v V
	return v, false
}

func (m *VecMap[K, V]) Has(key K) bool {
	_, ok := m.Get(key)
	return ok
}

func (m *VecMap[K, V]) Items() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m.mapv != nil {
			for k, v := range m.mapv {
				if !yield(k, v) {
					return
				}
			}
		} else {
			for _, item := range m.vec {
				if item.ok {
					if !yield(item.key, item.value) {
						return
					}
				}
			}
		}
	}
}

func (m *VecMap[K, V]) Set(key K, value V) {
	if m.mapv != nil {
		m.mapv[key] = value
		return
	}

	var freeitem *_VecMapPair[K, V]
	for i := range len(m.vec) {
		item := &m.vec[i]
		if item.ok && item.key == key {
			item.value = value
			return
		}
		if freeitem == nil && !item.ok {
			freeitem = item
		}
	}
	if freeitem != nil {
		freeitem.key = key
		freeitem.value = value
		freeitem.ok = true
		m.vecsize++
		return
	}

	if len(m.vec) >= m.factor {
		m.mapv = make(map[K]V, m.factor*2)
		for _, item := range m.vec {
			if item.ok {
				m.mapv[item.key] = item.value
			}
		}
		m.mapv[key] = value
		m.vec = nil
		m.vecsize = 0
		return
	}

	m.vec = append(m.vec, _VecMapPair[K, V]{
		key:   key,
		value: value,
		ok:    true,
	})
	m.vecsize++
}

func (m *VecMap[K, V]) Delete(key K) {
	if m.mapv != nil {
		delete(m.mapv, key)
		return
	}
	for i := range len(m.vec) {
		item := &m.vec[i]
		if item.ok && item.key == key {
			var empty _VecMapPair[K, V]
			*item = empty
			m.vecsize--
			break
		}
	}
}

func (m *VecMap[K, V]) Clear() {
	if m.mapv != nil {
		m.mapv = nil
	}
	m.vec = make([]_VecMapPair[K, V], 0, m.factor)
	m.vecsize = 0
}
