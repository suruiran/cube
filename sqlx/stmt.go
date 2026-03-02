package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/suruiran/cube/rbc"
)

type Stmt struct {
	*sql.Stmt
	rawsql          string
	params          []string
	lognames        []string
	fastargscache   atomic.Pointer[_FastArgsCache]
	fastresultcache atomic.Pointer[_FastResultCache]
}

type _FastArgsCache struct {
	argstype reflect.Type
	shape    *_ArgsShape
}

type _FastResultCache struct {
	eletype reflect.Type
	shape   *_ResultShape
}

var (
	ParamsRegexp = regexp.MustCompile(`\$\{\s*[a-zA-Z_]+[a-zA-Z_0-9]*\s*\}`)
)

func parseParamName(sql string) string {
	begin := strings.IndexByte(sql, '{')
	end := strings.LastIndexByte(sql, '}')
	return strings.TrimSpace(sql[begin+1 : end])
}

func (stmt *Stmt) prepare(ctx context.Context, rawsql string) error {
	dialect := MustDialect(ctx)
	idx := -1
	sqlbs := ParamsRegexp.ReplaceAllFunc([]byte(rawsql), func(m []byte) []byte {
		idx++
		paramname := parseParamName(string(m))
		stmt.params = append(stmt.params, paramname)
		stmt.lognames = append(stmt.lognames, fmt.Sprintf("arg: %s", paramname))
		return []byte(dialect.ParamPlaceholder(idx))
	})

	_sql := strings.TrimSpace(string(sqlbs))
	Logger().DebugContext(ctx, "Prepare Stmt", slog.String("sql", _sql))
	stdstmt, err := MustDB(ctx).PrepareContext(ctx, _sql)
	if err != nil {
		return err
	}
	stmt.Stmt = stdstmt
	stmt.rawsql = rawsql
	return nil
}

func NewStmt(ctx context.Context, rawsql string) (*Stmt, error) {
	stmt := &Stmt{}
	if err := stmt.prepare(ctx, rawsql); err != nil {
		return nil, err
	}
	return stmt, nil
}

type _ArgsShape struct {
	ParamIdx []int
	Fields   []*rbc.FieldWithTag
}

type _StmtArgsShapeKey struct {
	stmt *sql.Stmt
	typ  reflect.Type
}

var stmtArgsShapeCache sync.Map

func argsShapeOfStmt(eletype reflect.Type, stmt *Stmt) (*_ArgsShape, error) {
	cachekey := _StmtArgsShapeKey{
		stmt: stmt.Stmt,
		typ:  eletype,
	}
	val, ok := stmtArgsShapeCache.Load(cachekey)
	if ok {
		shape := val.(*_ArgsShape)
		return shape, nil
	}

	typeinfo := rbc.InfoOf(eletype)
	shape := &_ArgsShape{}

	for idx, param := range stmt.params {
		field := typeinfo.MustField("args", param)
		if field == nil {
			continue
		}
		shape.Fields = append(shape.Fields, field)
		shape.ParamIdx = append(shape.ParamIdx, idx)
	}

	if len(shape.Fields) < 1 {
		return nil, fmt.Errorf("sqlx: no field found for param `%s`", eletype.String())
	}

	stmtArgsShapeCache.Store(cachekey, shape)
	return shape, nil
}

func (s *Stmt) args(vv reflect.Value, args []any, updatefast bool) error {
	var valuptr unsafe.Pointer

	valtype := vv.Type()
	if valtype.Kind() == reflect.Pointer {
		valtype = valtype.Elem()
		if !vv.IsValid() || vv.IsNil() {
			return nil
		}
		valuptr = vv.UnsafePointer()
	} else {
		valuptr = vv.Addr().UnsafePointer()
	}

	var shape *_ArgsShape
	cache := s.fastargscache.Load()
	if cache != nil && cache.argstype == valtype && cache.shape != nil {
		shape = cache.shape
	} else {
		var err error
		shape, err = argsShapeOfStmt(valtype, s)
		if err != nil {
			return err
		}
		if updatefast && (cache == nil || cache.argstype != valtype) {
			cache = &_FastArgsCache{
				argstype: valtype,
				shape:    shape,
			}
			s.fastargscache.Store(cache)
		}
	}
	for i := range len(shape.Fields) {
		idx := shape.ParamIdx[i]
		field := shape.Fields[i]
		args[idx] = field.GetValue(valuptr)
	}
	return nil
}

var (
	nilargv = new(1)
)

func (s *Stmt) Args(vals ...any) ([]any, error) {
	if len(s.params) == 0 {
		return nil, nil
	}

	args := make([]any, len(s.params))
	for i := range len(s.params) {
		args[i] = nilargv
	}

	updatefast := len(vals) == 1
	for _, ele := range vals {
		vv := reflect.ValueOf(ele)
		if err := s.args(vv, args, updatefast); err != nil {
			return nil, err
		}
	}

	for i := range len(s.params) {
		if args[i] == nilargv {
			return nil, fmt.Errorf("sqlx: param `%s` is not set", s.params[i])
		}
	}
	return args, nil
}

func (s *Stmt) log(ctx context.Context, msg string, err error, begin time.Time, argsv []any) {
	logitem := _StmtLogItem{
		stmt:  s,
		args:  argsv,
		begin: begin,
	}

	if err == nil {
		Logger().DebugContext(
			ctx,
			msg,
			"stmt", logitem,
		)
		return
	}
	Logger().ErrorContext(
		ctx,
		msg,
		"stmt", logitem,
		slog.Any("error", err),
	)
}

func (s *Stmt) realstmt(ctx context.Context) *sql.Stmt {
	tx := PeekTx(ctx)
	if tx != nil {
		return tx.StmtContext(ctx, s.Stmt)
	}
	return s.Stmt
}

func (s *Stmt) Rows(ctx context.Context, args ...any) (*sql.Rows, error) {
	argsv, err := s.Args(args...)
	if err != nil {
		return nil, err
	}
	begin := time.Now()
	var rows *sql.Rows
	defer func() {
		s.log(ctx, "Stmt.Rows", err, begin, argsv)
	}()
	rows, err = s.realstmt(ctx).QueryContext(ctx, argsv...)
	return rows, nil
}

func (s *Stmt) MustRows(ctx context.Context, args ...any) *sql.Rows {
	rows, err := s.Rows(ctx, args...)
	if err != nil {
		panic(err)
	}
	return rows
}

func (s *Stmt) Exec(ctx context.Context, args ...any) (sql.Result, error) {
	argsv, err := s.Args(args...)
	if err != nil {
		return nil, err
	}
	begin := time.Now()

	var res sql.Result
	defer func() {
		s.log(ctx, "Stmt.Exec", err, begin, argsv)
	}()
	res, err = s.realstmt(ctx).ExecContext(ctx, argsv...)
	return res, err
}

func (s *Stmt) ExecContext(ctx context.Context, args ...any) (sql.Result, error) {
	return s.Exec(ctx, args...)
}

func (s *Stmt) MustExec(ctx context.Context, args ...any) sql.Result {
	res, err := s.Exec(ctx, args...)
	if err != nil {
		panic(err)
	}
	return res
}
