package rbc

import (
	"reflect"
	"sync"
)

type LazyField struct {
	name string
	*FieldWithTag
}

type FieldGroup struct {
	lock    sync.Mutex
	tagname string
	items   []*LazyField
}

func (fg *FieldGroup) Field(name string) *LazyField {
	lf := &LazyField{
		name: name,
	}
	fg.lock.Lock()
	defer fg.lock.Unlock()
	fg.items = append(fg.items, lf)
	return lf
}

func (fg *FieldGroup) fill(info *TypeInfo) {
	fg.lock.Lock()
	defer fg.lock.Unlock()

	for _, lf := range fg.items {
		lf.FieldWithTag = info.MustField(fg.tagname, lf.name)
	}
}

var (
	lazyfgslock sync.Mutex
	lazyfgs     = map[reflect.Type]*FieldGroup{}
)

func NewFieldGroup[T any](tagname string) *FieldGroup {
	lazyfgslock.Lock()
	defer lazyfgslock.Unlock()

	stype := reflect.TypeFor[T]()
	pv, ok := lazyfgs[stype]
	if ok {
		return pv
	}
	cv := &FieldGroup{
		tagname: tagname,
	}
	lazyfgs[stype] = cv
	return cv
}
