package logic

import (
	"fmt"
	"math"
	"reflect"
	"slices"
)

func _tobool(val any, derefcount int, rawval any) bool {
	switch val := val.(type) {
	case string:
		return val != ""
	case LazyBool:
		{
			return val.Bool()
		}
	case *LazyBool:
		{
			return val.Bool()
		}
	case bool:
		return val
	case func() bool:
		return val()
	case nil:
		return false
	}

	vv := reflect.ValueOf(val)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		{
			return vv.Int() != 0
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		{
			return vv.Uint() != 0
		}
	case reflect.Float32, reflect.Float64:
		{
			f := vv.Float()
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return false
			}
			return f != 0
		}
	case reflect.Pointer:
		{
			if vv.IsNil() {
				return false
			}
			if derefcount > 2 {
				panic(fmt.Errorf("cube: unexpected pointer, `%v`", rawval))
			}
			return _tobool(vv.Elem().Interface(), derefcount+1, rawval)
		}
	case reflect.Slice, reflect.Map:
		{
			return vv.Len() != 0
		}
	default:
		{
			panic(fmt.Errorf("cube: can not converted to bool, can pass a `func() bool`, `%v`", rawval))
		}
	}
}

func tobool(val any) bool {
	return _tobool(val, 0, val)
}

func All(vals ...any) bool {
	for _, v := range vals {
		if !tobool(v) {
			return false
		}
	}
	return true
}

func Any(vals ...any) bool {
	return slices.ContainsFunc(vals, tobool)
}

func None(vals ...any) bool {
	return !slices.ContainsFunc(vals, tobool)
}
