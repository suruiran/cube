package rbc

import (
	"database/sql"
	"reflect"
	"unsafe"
)

var (
	ptrcasts     = map[reflect.Type]func(unsafe.Pointer) any{}
	derefs       = map[reflect.Type]func(unsafe.Pointer) any{}
	sets         = map[reflect.Type]func(unsafe.Pointer, any){}
	constructors = map[reflect.Type]func() (any, unsafe.Pointer){}
	memclrs      = map[reflect.Type]func(unsafe.Pointer){}
)

func registeronetype[T any]() {
	typ := reflect.TypeFor[T]()
	ptrcasts[typ] = func(vv unsafe.Pointer) any { return (*T)(vv) }
	derefs[typ] = func(vv unsafe.Pointer) any { return *((*T)(vv)) }
	sets[typ] = func(vv unsafe.Pointer, fv any) {
		*((*T)(vv)) = fv.(T)
	}
	memclrs[typ] = func(vv unsafe.Pointer) {
		var zero T
		*((*T)(vv)) = zero
	}
	constructors[typ] = func() (any, unsafe.Pointer) {
		val := new(T)
		return val, unsafe.Pointer(val)
	}
}

func RegisterType[T any](tags ...string) {
	typ := reflect.TypeFor[T]()
	pkgpath := typ.PkgPath()

	switch typ.Kind() {
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Bool, reflect.String, reflect.Struct, reflect.Complex64, reflect.Complex128:
		{
			registeronetype[T]()
			registeronetype[*T]()
			if pkgpath != "database/sql" {
				registeronetype[sql.Null[T]]()
			}
		}
	default:
		{
			registeronetype[T]()
		}
	}

	tags = append(tags, "db", "sql", "args")
	if typ.Kind() == reflect.Struct {

		switch pkgpath {
		case "time", "database/sql":
		default:
			{
				typeinfo := InfoFor[T]()
				for _, tagname := range tags {
					_ = typeinfo.inittag(tagname)
				}
			}
		}
	}
}

func SpellSteal[T any](tags ...string) {
	RegisterType[T](tags...)
}
