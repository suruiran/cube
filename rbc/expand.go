package rbc

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"
)

func npush[T any](eles []T, vs ...T) []T {
	c := make([]T, len(eles))
	copy(c, eles)
	return append(c, vs...)
}

func expand(vv reflect.Value, top *TypeInfo, indexes []int) *TypeInfo {
	if vv.Kind() != reflect.Struct {
		panic(fmt.Sprintf("sqlx.expand: T must be a struct, got `%T`", vv.Type()))
	}

	registerBuiltins()

	if top == nil {
		top = &TypeInfo{
			Type: vv.Type(),
		}
	}

	for fv := range vv.Type().Fields() {
		if !fv.IsExported() {
			continue
		}

		loc_index := npush(indexes, fv.Index...)

		offset := fieldoffset(top.Type, loc_index)

		ff := &Field{
			Info:     fv,
			Index:    loc_index,
			ptrcast:  ptrcasts[fv.Type],
			ptrderef: derefs[fv.Type],
			set:      sets[fv.Type],
			jump:     func(vv unsafe.Pointer) unsafe.Pointer { return unsafe.Add(vv, offset) },
			jump_with_aegis: func(crx context.Context, vv unsafe.Pointer) (unsafe.Pointer, error) {
				return unsafe.Add(vv, offset), nil
			},
		}

		if fv.Anonymous {
			if fv.Type.Kind() == reflect.Struct {
				expand(vv.FieldByIndex(fv.Index), top, npush(loc_index))
			} else {
				subinfo := InfoOf(fv.Type.Elem())
				for _, f := range subinfo.fields {
					cf := new(Field)
					*cf = *f
					blink(cf, ff)
					blink_with_aegis(top, cf, ff)
					top.append(cf)
				}
			}
			ff.nested = true
			top.nesteds = append(top.nesteds, ff)
			continue
		}
		top.append(ff)
	}
	return top
}

// blink
// Avoid nested pointers whenever possible. this function can not be safe.
//
// `Never blink into the fog of war` -- DOTA community
func blink(cf *Field, pf *Field) {
	cjump := cf.jump
	cf.jump = func(ppp unsafe.Pointer) unsafe.Pointer {
		pp := pf.jump(ppp)
		return cjump(*(*unsafe.Pointer)(pp))
	}
}

type _OnDeferNilCtx struct {
	context.Context
	toptype *TypeInfo
	field   *Field
}

type DerefNilContext interface {
	context.Context
	Top() *TypeInfo
	Field() *Field
}

func WithOnDerefNil(ctx context.Context, fn func(ctx DerefNilContext, eletype reflect.Type) (unsafe.Pointer, error)) context.Context {
	return context.WithValue(ctx, _CtxKeyOnDerefNil, fn)
}

var _ DerefNilContext = (*_OnDeferNilCtx)(nil)

func (ctx *_OnDeferNilCtx) Top() *TypeInfo {
	return ctx.toptype
}

func (ctx *_OnDeferNilCtx) Field() *Field {
	return ctx.field
}

func blink_with_aegis(typeinfo *TypeInfo, cf *Field, pf *Field) {
	cjumpwa := cf.jump_with_aegis
	pfeletype := pf.Info.Type.Elem()
	cf.jump_with_aegis = func(ctx context.Context, ppp unsafe.Pointer) (unsafe.Pointer, error) {
		// pp: **T
		pp, err := pf.jump_with_aegis(ctx, ppp)
		if err != nil {
			return nil, err
		}
		// p: *T
		p := *(*unsafe.Pointer)(pp)
		if p == nil {
			if fn := ctx.Value(_CtxKeyOnDerefNil); fn != nil {
				// alloc T
				_p, err := (fn.(func(ctx DerefNilContext, eletype reflect.Type) (unsafe.Pointer, error)))(
					&_OnDeferNilCtx{
						Context: ctx,
						toptype: typeinfo,
						field:   cf,
					},
					pfeletype,
				)
				if err != nil {
					return nil, err
				}
				*(*unsafe.Pointer)(pp) = _p
				p = _p
			}
		}
		return cjumpwa(ctx, p)
	}
}
