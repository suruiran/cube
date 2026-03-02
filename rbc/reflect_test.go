package rbc

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

type CreatedAtModel struct {
	CreatedAt int64 `db:"created_at"`
}

type CommonModel struct {
	CreatedAtModel
	UpdatedAt int64  `db:"updated_at"`
	DeletedAt *int64 `db:"deleted_at"`
}

type A struct {
	A1 int64 `db:"a1"`
}

type B struct {
	A
	B1 string
}

type C struct {
	*B
	C1 int64
}

type User struct {
	Id   int64  `db:"id"`
	Name string `db:"name"`
	CommonModel
	*C
}

func newobj() (*User, unsafe.Pointer) {
	obj := &User{
		Id:   1,
		Name: "user1",
		CommonModel: CommonModel{
			CreatedAtModel: CreatedAtModel{
				CreatedAt: 100,
			},
			UpdatedAt: 2,
			DeletedAt: new(int64(33)),
		},
		C: &C{
			B: &B{
				A: A{
					A1: 1000,
				},
				B1: "b1",
			},
			C1: 2000,
		},
	}
	return obj, unsafe.Pointer(obj)
}

var (
	fieldgroup     = NewFieldGroup[User]("db")
	UpdatedAtField = fieldgroup.Field("updated_at")
	A1Field        = fieldgroup.Field("a1")
)

func init() {
	RegisterType[A]()
	RegisterType[B]()
	RegisterType[C]()
	RegisterType[User]()
}

func TestUserFields(t *testing.T) {
	_, uptr := newobj()

	for _, f := range InfoFor[User]().fields {
		fmt.Println(f.Info.Name, f.GetValue(uptr), f.GetPtr(uptr))
	}

	stackobj := User{}
	stackobj.UpdatedAt = 122

	fmt.Println(UpdatedAtField.GetValue(unsafe.Pointer(&stackobj)))
}

func TestGetValueWithAegis(t *testing.T) {
	var obj User
	obj.UpdatedAt = 121
	fmt.Println(UpdatedAtField.GetValue(unsafe.Pointer(&obj)))

	val, err := A1Field.GetValueWithAegis(
		WithOnDerefNil(
			t.Context(),
			func(ctx DerefNilContext, eletype reflect.Type) (unsafe.Pointer, error) {
				switch eletype {
				case reflect.TypeFor[B]():
					{
						tmp := new(B)
						tmp.A1 = 12121
						return unsafe.Pointer(tmp), nil
					}
				case reflect.TypeFor[C]():
					{
						tmp := new(C)
						return unsafe.Pointer(tmp), nil
					}
				}
				return unsafe.Pointer(nil), fmt.Errorf("unexpected type %s", eletype.String())
			},
		),
		unsafe.Pointer(&obj),
	)
	fmt.Println(val, err)
}

func TestPool(t *testing.T) {
	_ = WithArena(t.Context(), func(ctx context.Context) error {
		var obj User
		obj.UpdatedAt = 121

		fmt.Println(UpdatedAtField.GetValue(unsafe.Pointer(&obj)))
		val, err := A1Field.GetValueWithAegis(
			ctx,
			unsafe.Pointer(&obj),
		)
		fmt.Println(val, err)
		fmt.Println(obj.C)
		obj.A1 = 12345
		return nil
	})

	_ = WithArena(t.Context(), func(ctx context.Context) error {
		var obj User
		obj.UpdatedAt = 127
		fmt.Println(UpdatedAtField.GetValue(unsafe.Pointer(&obj)))

		val, err := A1Field.GetValueWithAegis(
			ctx,
			unsafe.Pointer(&obj),
		)
		fmt.Println(val, err)
		fmt.Println(obj.C, obj.A1)
		return nil
	})
}

func BenchmarkStdReflectGetValue(b *testing.B) {
	obj, _ := newobj()

	vv := reflect.ValueOf(obj).Elem()

	b.ResetTimer()
	for b.Loop() {
		vv.FieldByIndex(UpdatedAtField.Index).Interface()
	}
}

func BenchmarkStdReflectGetPtr(b *testing.B) {
	obj, _ := newobj()

	vv := reflect.ValueOf(obj).Elem()

	b.ResetTimer()
	for b.Loop() {
		vv.FieldByIndex(UpdatedAtField.Index).Addr().Interface()
	}
}

func BenchmarkGetValue(b *testing.B) {
	_, uptr := newobj()

	b.ResetTimer()
	for b.Loop() {
		UpdatedAtField.GetValue(uptr)
	}
}

func BenchmarkGetPtr(b *testing.B) {
	_, objuptr := newobj()
	b.ResetTimer()
	for b.Loop() {
		UpdatedAtField.GetPtr(objuptr)
	}
}

func BenchmarkGetValueWithAegis(b *testing.B) {
	_, uptr := newobj()
	b.ResetTimer()
	for b.Loop() {
		UpdatedAtField.GetValueWithAegis(b.Context(), uptr) //nolint:errcheck
	}
}

func BenchmarkGetPtrWithAegis(b *testing.B) {
	_, objuptr := newobj()

	for b.Loop() {
		UpdatedAtField.GetPtrWithAegis(b.Context(), objuptr) //nolint:errcheck
	}
}
