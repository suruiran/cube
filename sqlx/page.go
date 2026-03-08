package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"unsafe"
)

func pageinternal[T any](ctx context.Context, stmt *Stmt, rows *sql.Rows, page []T, filltmp func(shape *_ResultShape, idx int, tmp *[]any)) ([]T, bool, error) {
	eletype := reflect.TypeFor[T]()
	shape, err := resultShapeOfStmt(eletype, stmt, rows)
	if err != nil {
		return nil, true, err
	}

	pagesize := len(page)
	if pagesize < 1 {
		return nil, true, fmt.Errorf("sqlx: page size must be greater than 0")
	}

	var tmp = shape.mktmp()
	size := 0

	for rows.Next() {
		filltmp(shape, size, &tmp)
		if err := rows.Scan(tmp...); err != nil {
			return nil, true, err
		}
		size++
		if pagesize > 0 && size >= pagesize {
			return page[:size], true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return page[:size], false, nil
}

func Page[T any](ctx context.Context, stmt *Stmt, rows *sql.Rows, page []T) ([]T, bool, error) {
	return pageinternal(
		ctx, stmt, rows, page,
		func(shape *_ResultShape, idx int, tmp *[]any) {
			ptr := &page[idx]
			shape.filltempunsafe(unsafe.Pointer(ptr), tmp)
		},
	)
}

type _PageFn[T any] func(ctx context.Context, stmt *Stmt, rows *sql.Rows, page []T) ([]T, bool, error)

func firstinternal[T any](ctx context.Context, stmt *Stmt, page _PageFn[T], args ...any) (*T, error) {
	rows, err := stmt.Rows(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // nolint: errcheck

	var lst = make([]T, 1)
	tmp, _, err := page(ctx, stmt, rows, lst)
	if err != nil {
		return nil, err
	}
	return &tmp[0], nil
}

func First[T any](ctx context.Context, stmt *Stmt, args ...any) (*T, error) {
	return firstinternal(ctx, stmt, Page[T], args...)
}

func allinternal[T any](ctx context.Context, stmt *Stmt, sizehint int, page _PageFn[T], args ...any) ([]T, error) {
	if sizehint < 1 {
		return nil, fmt.Errorf("sqlx: sizehint must be greater than 0")
	}

	rows, err := stmt.Rows(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // nolint: errcheck

	lst := make([]T, sizehint)
	cursor := 0
	for {
		if cursor >= len(lst) {
			lst = append(lst, make([]T, sizehint)...)
		}
		tmp, more, err := page(ctx, stmt, rows, lst[cursor:])
		if err != nil {
			return nil, err
		}
		cursor += len(tmp)
		if !more {
			break
		}
	}
	return lst[:cursor], nil
}

func All[T any](ctx context.Context, stmt *Stmt, sizehint int, args ...any) ([]T, error) {
	return allinternal(ctx, stmt, sizehint, Page[T], args...)
}
