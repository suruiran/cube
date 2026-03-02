package rbc

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type Field struct {
	Info  reflect.StructField
	Index []int

	nested   bool
	ptrcast  func(unsafe.Pointer) any
	ptrderef func(unsafe.Pointer) any
	set      func(unsafe.Pointer, any)

	jump            func(unsafe.Pointer) unsafe.Pointer
	jump_with_aegis func(context.Context, unsafe.Pointer) (unsafe.Pointer, error)
}

type FieldWithTag struct {
	*Field
	Tag *Tag
}

type TypeInfo struct {
	Type    reflect.Type
	nesteds []*Field
	fields  []*Field

	tagsmap map[string]map[string]*FieldWithTag
	tagslst map[string][]*FieldWithTag
}

func (ti *TypeInfo) append(f *Field) {
	ti.fields = append(ti.fields, f)
}

func (ti *TypeInfo) Field(tagname, name string) (*FieldWithTag, error) {
	if err := ti.inittag(tagname); err != nil {
		return nil, err
	}
	fv, ok := ti.tagsmap[tagname][name]
	if !ok {
		return nil, fmt.Errorf("sqlx: field `%s` not found in type `%s`", name, ti.Type.Name())
	}
	return fv, nil
}

func (ti *TypeInfo) MustField(tagname, name string) *FieldWithTag {
	fv, err := ti.Field(tagname, name)
	if err != nil {
		panic(err)
	}
	return fv
}

func (ti *TypeInfo) Fields(tagname string) ([]*FieldWithTag, error) {
	if err := ti.inittag(tagname); err != nil {
		return nil, err
	}
	lst := ti.tagslst[tagname]
	if lst == nil {
		return nil, fmt.Errorf("sqlx: tag `%s` not found in type `%s`", tagname, ti.Type.String())
	}
	return lst, nil
}

func (ti *TypeInfo) MustFields(tagname string) []*FieldWithTag {
	lst, err := ti.Fields(tagname)
	if err != nil {
		panic(err)
	}
	return lst
}

var (
	typeinfocache = sync.Map{}
)

func InfoOf(t reflect.Type) *TypeInfo {
	if ti, ok := typeinfocache.Load(t); ok {
		return ti.(*TypeInfo)
	}
	ti := expand(reflect.ValueOf(anchor(t)).Elem(), nil, nil)

	lazyfgslock.Lock()
	fg, ok := lazyfgs[t]
	lazyfgslock.Unlock()
	if ok {
		fg.fill(ti)
	}
	typeinfocache.Store(t, ti)
	return ti
}

func InfoFor[T any]() *TypeInfo {
	return InfoOf(reflect.TypeFor[T]())
}
