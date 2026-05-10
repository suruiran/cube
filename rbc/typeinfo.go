package rbc

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

type Field struct {
	Info  reflect.StructField
	Index []int

	parent  reflect.Type
	metakey string

	offset   uintptr
	nested   bool
	ptrcast  func(unsafe.Pointer) any
	ptrderef func(unsafe.Pointer) any
	set      func(unsafe.Pointer, any)

	jump            func(unsafe.Pointer) unsafe.Pointer
	jump_with_aegis func(context.Context, unsafe.Pointer) (unsafe.Pointer, error)
}

func (fi *Field) Nested() *TypeInfo {
	if !fi.nested {
		panic(fmt.Errorf("cube.rbc: field is not anonymous"))
	}
	return InfoOf(fi.Info.Type.Elem())
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
		return nil, fmt.Errorf("cube.rbc: field `%s` not found in type `%s`", name, ti.Type.Name())
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
		return nil, fmt.Errorf("cube.rbc: tag `%s` not found in type `%s`", tagname, ti.Type.String())
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
	typeinfocache _StaticCache
)

func InfoOf(t reflect.Type) *TypeInfo {
	val, err := typeinfocache.GetOrCompute(t.String(), func() (any, error) {
		ti := expand(t, reflect.ValueOf(AnchorOf(t)).Elem(), nil, nil)

		lazyfgslock.Lock()
		fg, ok := lazyfgs[t]
		lazyfgslock.Unlock()
		if ok {
			fg.fill(ti)
		}
		return ti, nil
	})
	if err != nil {
		if errors.Is(err, errReEntered) {
			panic(fmt.Sprintf("cube.rbc: type `%s` is recursive", t.String()))
		}
		panic(err)
	}
	return val.(*TypeInfo)
}

func InfoFor[T any]() *TypeInfo {
	return InfoOf(reflect.TypeFor[T]())
}
