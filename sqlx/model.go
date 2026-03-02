package sqlx

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/suruiran/cube/rbc"
)

type ModelField struct {
	Name   string
	GoType reflect.Type
	Opts   rbc.TagOptions
}

type ModelInfo struct {
	GoType reflect.Type
	Fields []*ModelField
}

type IDialect interface {
	EnsureTables(ctx context.Context, infos []*ModelInfo) error

	ParamPlaceholder(index int) string
}

var _ensurebackedup = true

func DisableEnsureBackedUp() {
	_ensurebackedup = false
}

func Sync(ctx context.Context) error {
	if _ensurebackedup {
		envkey := fmt.Sprintf("SQLX_SYNC_%s", time.Now().Format("06010215"))
		if os.Getenv(envkey) != "ALREADY_BACKED_UP" {
			return fmt.Errorf(
				"sqlx: migrate might be destructive! please ensure you have backed up, and then `export %s=ALREADY_BACKED_UP`",
				envkey,
			)
		}
	}

	var tableinfos []*ModelInfo
	models.Range(func(key, value any) bool {
		tableinfos = append(tableinfos, value.(*ModelInfo))
		return true
	})
	dialect := MustDialect(ctx)
	return dialect.EnsureTables(ctx, tableinfos)
}

var (
	models sync.Map
)

func AsModel[T any]() {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("sqlx: AsModel only support struct type, but got %s", typ.String()))
	}
	if _, ok := models.Load(typ); ok {
		return
	}

	typeinfo := rbc.InfoOf(typ)

	// pre-hot for `db`
	_, _ = typeinfo.Fields("db")

	fields := typeinfo.MustFields("sql")

	tableinfo := &ModelInfo{
		GoType: typ,
		Fields: make([]*ModelField, 0, len(fields)),
	}

	for _, fv := range fields {
		col := &ModelField{
			Name:   fv.Tag.Name,
			GoType: fv.Info.Type,
			Opts:   fv.Tag.Opts,
		}
		tableinfo.Fields = append(tableinfo.Fields, col)
	}

	models.Store(typ, tableinfo)
}
