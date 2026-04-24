package rbc

import (
	"context"
	"reflect"
	"unsafe"
)

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
