package rbc

import (
	"sync"
)

type Meta struct {
	Lock sync.Mutex
	Vals map[any]any
}

var (
	allmetas sync.Map
)

func (fi *Field) Meta() *Meta {
	prev, ok := allmetas.Load(fi.metakey)
	if ok {
		return prev.(*Meta)
	}
	ptr := new(Meta)
	prev, loaded := allmetas.LoadOrStore(fi.metakey, ptr)
	if loaded {
		return prev.(*Meta)
	}
	return ptr
}

func MetaFor[T any, F any](fnc func(anchor *T) *F, filters ...func(fi *Field) bool) *Meta {
	return MustFieldFor(fnc, filters...).Meta()
}
