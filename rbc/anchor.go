package rbc

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	anchors sync.Map
)

func anchor(tv reflect.Type) any {
	if tv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Anchor: T must be a struct, got `%T`", tv))
	}
	prev, ok := anchors.Load(tv)
	if ok {
		return prev
	}
	ptr := reflect.New(tv).Interface()
	anchors.Store(tv, ptr)
	return ptr
}

func fieldoffset(rv reflect.Type, indexes []int) uintptr {
	aptr := anchor(rv)
	aptrv := reflect.ValueOf(aptr)
	auptr := aptrv.UnsafePointer()

	fvv := aptrv.Elem().FieldByIndex(indexes)
	fvvptr := fvv.Addr().UnsafePointer()
	return uintptr(fvvptr) - uintptr(auptr)
}
