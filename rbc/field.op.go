package rbc

import (
	"context"
	"database/sql"
	"reflect"
	"unsafe"
)

var (
	ptrcasts     = map[reflect.Type]func(unsafe.Pointer) any{}
	derefs       = map[reflect.Type]func(unsafe.Pointer) any{}
	sets         = map[reflect.Type]func(unsafe.Pointer, any){}
	constructors = map[reflect.Type]func() (any, unsafe.Pointer){}
)

func registeronetype[T any]() {
	typ := reflect.TypeFor[T]()
	ptrcasts[typ] = func(vv unsafe.Pointer) any { return (*T)(vv) }
	derefs[typ] = func(vv unsafe.Pointer) any { return *((*T)(vv)) }
	sets[typ] = func(vv unsafe.Pointer, fv any) {
		*((*T)(vv)) = fv.(T)
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

func (f *Field) GetPtr(vv unsafe.Pointer) any {
	fuptr := f.jump(vv)
	if f.ptrcast != nil {
		return f.ptrcast(fuptr)
	}
	on_std_reflect_call(f.Info.Type, _LogItemPtrcast)
	return reflect.NewAt(f.Info.Type, fuptr).Interface()
}

func (f *Field) GetPtrWithAegis(ctx context.Context, vv unsafe.Pointer) (any, error) {
	fuptr, err := f.jump_with_aegis(ctx, vv)
	if err != nil {
		return nil, err
	}
	if f.ptrcast != nil {
		return f.ptrcast(fuptr), nil
	}
	on_std_reflect_call(f.Info.Type, _LogItemPtrcast)
	return reflect.NewAt(f.Info.Type, fuptr).Interface(), nil
}

func (f *Field) GetValue(vv unsafe.Pointer) any {
	fuptr := f.jump(vv)
	if f.ptrderef != nil {
		return f.ptrderef(fuptr)
	}
	on_std_reflect_call(f.Info.Type, _LogItemDeref)
	return reflect.NewAt(f.Info.Type, fuptr).Elem().Interface()
}

func (f *Field) GetValueWithAegis(ctx context.Context, vv unsafe.Pointer) (any, error) {
	fuptr, err := f.jump_with_aegis(ctx, vv)
	if err != nil {
		return nil, err
	}
	if f.ptrderef != nil {
		return f.ptrderef(fuptr), nil
	}
	on_std_reflect_call(f.Info.Type, _LogItemDeref)
	return reflect.NewAt(f.Info.Type, fuptr).Elem().Interface(), nil
}

func (f *Field) Set(vv unsafe.Pointer, fv any) {
	fuptr := f.jump(vv)
	if f.set != nil {
		f.set(fuptr, fv)
		return
	}
	on_std_reflect_call(f.Info.Type, _LogItemSet)
	reflect.NewAt(f.Info.Type, fuptr).Elem().Set(reflect.ValueOf(fv))
}

func (f *Field) SetWithAegis(ctx context.Context, vv unsafe.Pointer, fv any) error {
	fuptr, err := f.jump_with_aegis(ctx, vv)
	if err != nil {
		return err
	}
	if f.set != nil {
		f.set(fuptr, fv)
		return nil
	}
	on_std_reflect_call(f.Info.Type, _LogItemSet)
	reflect.NewAt(f.Info.Type, fuptr).Elem().Set(reflect.ValueOf(fv))
	return nil
}
