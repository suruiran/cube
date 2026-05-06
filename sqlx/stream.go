package sqlx

import (
	"context"
	"database/sql"
	"iter"
	"reflect"
	"unsafe"
)

func streaminternal[T any](
	ctx context.Context, stmt *Stmt, rows *sql.Rows,
	initemp func(shape *_ResultShape, tmp *[]any),
	filltmp func(shape *_ResultShape, tmp *[]any) *T,
) iter.Seq2[*T, error] {
	eletype := reflect.TypeFor[T]()
	return func(yield func(*T, error) bool) {
		shape, err := resultShapeOfStmt(eletype, stmt, rows)
		if err != nil {
			yield(nil, err)
			return
		}
		var tmp = shape.mktmp()
		initemp(shape, &tmp)

		cc := 0
		for rows.Next() {
			cc++
			if cc%10 == 0 {
				if err := ctx.Err(); err != nil {
					yield(nil, ctx.Err())
					return
				}
			}

			ptr := filltmp(shape, &tmp)

			if err := rows.Scan(tmp...); err != nil {
				yield(nil, err)
				return
			}
			if !yield(ptr, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, err)
			return
		}
	}
}

// Stream
// the yielded ptr reused for each row, caller can not store it.
func Stream[T any](ctx context.Context, stmt *Stmt, rows *sql.Rows) iter.Seq2[*T, error] {
	var val T
	ptr := &val
	return streaminternal(
		ctx, stmt, rows,
		func(shape *_ResultShape, tmp *[]any) {
			shape.filltempunsafe(unsafe.Pointer(ptr), tmp)
		},
		func(shape *_ResultShape, tmp *[]any) *T {
			return ptr
		},
	)
}

type MapOperator[T any, U any] struct {
	stmt *Stmt
	op   func(ctx context.Context, t *T) (U, error)
}

func (m *MapOperator[T, U]) Iter(ctx context.Context, args ...any) iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		rows, err := m.stmt.Rows(ctx, args...)
		if err != nil {
			var r U
			yield(r, err)
			return
		}
		defer rows.Close() //nolint:errcheck

		for t, err := range Stream[T](ctx, m.stmt, rows) {
			if err != nil {
				var r U
				yield(r, err)
				return
			}
			r, err := m.op(ctx, t)
			if err != nil {
				var r U
				yield(r, err)
				return
			}
			if !yield(r, nil) {
				return
			}
		}
	}
}

func NewMap[T any, U any](stmt *Stmt, operator func(ctx context.Context, t *T) (U, error)) *MapOperator[T, U] {
	return &MapOperator[T, U]{
		stmt: stmt,
		op:   operator,
	}
}

type ReduceOperator[T any, U any] struct {
	stmt *Stmt
	op   func(context.Context, U, *T) (U, error)
}

func (r *ReduceOperator[T, U]) Calc(ctx context.Context, result U, args ...any) (U, error) {
	rows, err := r.stmt.Rows(ctx, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close() //nolint:errcheck

	for t, err := range Stream[T](ctx, r.stmt, rows) {
		if err != nil {
			return result, err
		}
		result, err = r.op(ctx, result, t)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func NewReduce[T any, U any](stmt *Stmt, operator func(context.Context, U, *T) (U, error)) *ReduceOperator[T, U] {
	return &ReduceOperator[T, U]{
		stmt: stmt,
		op:   operator,
	}
}
