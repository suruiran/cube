package sqlx

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/suruiran/cube/rbc"
)

type _ResultShape struct {
	cols     []*sql.ColumnType
	names    []string
	types    []reflect.Type
	fields   []*rbc.FieldWithTag
	discards []any
}

func newdiscard(typ reflect.Type) any {
	return reflect.New(typ).Interface()
}

func (shape *_ResultShape) mktmp() []any {
	return make([]any, len(shape.cols))
}

func (shape *_ResultShape) filltempunsafe(dest unsafe.Pointer, tmp *[]any) {
	for idx := range len(shape.cols) {
		field := shape.fields[idx]
		if field == nil {
			(*tmp)[idx] = shape.discards[idx]
			continue
		}
		(*tmp)[idx] = field.GetPtr(dest)
	}
}

func mkshape(eletype reflect.Type, row *sql.Rows) (*_ResultShape, error) {
	cols, err := row.ColumnTypes()
	if err != nil {
		return nil, err
	}

	shape := &_ResultShape{
		cols:  cols,
		names: make([]string, 0, len(cols)),
		types: make([]reflect.Type, 0, len(cols)),
	}

	for _, col := range cols {
		shape.names = append(shape.names, col.Name())
		shape.types = append(shape.types, col.ScanType())
	}

	typeinf := rbc.InfoOf(eletype)

	size := 0
	for idx, name := range shape.names {
		fv := typeinf.MustField("db", name)
		shape.fields = append(shape.fields, fv)
		if fv != nil {
			size++
			shape.discards = append(shape.discards, nil)
		} else {
			shape.discards = append(shape.discards, newdiscard(shape.types[idx]))
		}
	}

	if size < 1 {
		return nil, fmt.Errorf("sqlx: no field found for %s", eletype.String())
	}
	return shape, nil
}

var stmtResultShapeCache sync.Map

type _StmtShapeKey struct {
	stmt *Stmt
	typ  reflect.Type
}

func resultShapeOfStmt(eletype reflect.Type, stmt *Stmt, row *sql.Rows) (*_ResultShape, error) {
	fastcache := stmt.fastresultcache.Load()
	if fastcache != nil && fastcache.eletype == eletype {
		return fastcache.shape, nil
	}

	cachekey := _StmtShapeKey{
		stmt: stmt,
		typ:  eletype,
	}
	val, ok := stmtResultShapeCache.Load(cachekey)
	if ok {
		return val.(*_ResultShape), nil
	}

	shape, err := mkshape(eletype, row)
	if err != nil {
		return nil, err
	}
	stmtResultShapeCache.Store(cachekey, shape)
	stmt.fastresultcache.Store(&_FastResultCache{
		eletype: eletype,
		shape:   shape,
	})
	return shape, nil
}
