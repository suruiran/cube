package rbc

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

var (
	anchors sync.Map
)

func AnchorOf(tv reflect.Type) any {
	if tv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Anchor: T must be a struct, got `%T`", tv))
	}
	prev, ok := anchors.Load(tv)
	if ok {
		return prev
	}
	ptr := reflect.New(tv).Interface()
	prev, loaded := anchors.LoadOrStore(tv, ptr)
	if loaded {
		return prev
	}
	return ptr
}

func AnchorFor[T any]() *T {
	return AnchorOf(reflect.TypeFor[T]()).(*T)
}

func fieldoffset(rv reflect.Type, indexes []int) uintptr {
	aptr := AnchorOf(rv)
	aptrv := reflect.ValueOf(aptr)
	auptr := aptrv.UnsafePointer()

	fvv := aptrv.Elem().FieldByIndex(indexes)
	fvvptr := fvv.Addr().UnsafePointer()
	return uintptr(fvvptr) - uintptr(auptr)
}

func FieldsFor[T any, F any](fnc func(anchor *T) *F, filters ...func(fi *Field) bool) []*Field {
	tt := reflect.TypeFor[T]()
	anchor := AnchorOf(tt).(*T)
	info := InfoOf(tt)
	fptr := fnc(anchor)
	ftype := reflect.TypeFor[F]()
	offset := uintptr(unsafe.Pointer(fptr)) - uintptr(unsafe.Pointer(anchor))

	feles := make([]*Field, 0, 10)

	for _, f := range info.fields {
		if f.offset == offset && ftype == f.Info.Type {
			feles = append(feles, f)
		}
	}
	for _, f := range info.nesteds {
		if f.offset == offset && ftype == f.Info.Type {
			feles = append(feles, f)
		}
	}

	if len(feles) < 1 {
		return nil
	}
	if len(filters) < 1 {
		return feles
	}
	remains := make([]*Field, 0, len(feles))
loop:
	for _, f := range feles {
		for _, filter := range filters {
			if !filter(f) {
				continue loop
			}
		}
		remains = append(remains, f)
	}
	return remains
}

func MustFieldFor[T any, F any](fnc func(anchor *T) *F, filters ...func(fi *Field) bool) *Field {
	feles := FieldsFor(fnc, filters...)
	if len(feles) != 1 {
		panic(fmt.Errorf(
			"cube.rbc: MustFieldFor[%s, %s] expected exactly 1 field, but found %d",
			reflect.TypeFor[T]().String(),
			reflect.TypeFor[F]().String(),
			len(feles),
		))
	}
	return feles[0]
}
