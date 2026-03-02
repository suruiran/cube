package sqlite

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/suruiran/cube/sqlx"
)

type _IndexInfo struct {
	Name string
	DDL  string
}

type _TableInfo struct {
	Name    string
	DDL     string
	Indexes []*_IndexInfo

	version string
}

func base62(bs []byte) string {
	var i big.Int
	i.SetBytes(bs)
	return i.Text(62)
}

func (info *_TableInfo) Version() string {
	if info.version == "" {
		sort.Slice(
			info.Indexes,
			func(i int, j int) bool {
				a := info.Indexes[i]
				b := info.Indexes[j]
				return a.Name < b.Name
			},
		)

		hobj := md5.New()
		buf := bufio.NewWriter(hobj)
		_, _ = buf.WriteString(info.Name)
		_, _ = buf.WriteString(info.DDL)

		for _, ele := range info.Indexes {
			_, _ = buf.WriteString(ele.Name)
			_, _ = buf.WriteString(ele.DDL)
		}
		_ = buf.Flush()
		info.version = base62(hobj.Sum(nil))
	}
	return info.version
}

func (info *_TableInfo) IndexNames() IndexNames {
	if len(info.Indexes) == 0 {
		return nil
	}
	names := make(IndexNames, 0, len(info.Indexes))
	for _, ele := range info.Indexes {
		names = append(names, ele.Name)
	}
	return names
}

func createTable(ctx context.Context, info *_TableInfo) error {
	exec := sqlx.MustExecutor(ctx)
	if _, err := exec.ExecContext(ctx, info.DDL); err != nil {
		return err
	}
	for _, ele := range info.Indexes {
		if _, err := exec.ExecContext(ctx, ele.DDL); err != nil {
			return err
		}
	}
	return insertModelLog(ctx, &_ModelLog{
		Name:    info.Name,
		Version: info.Version(),
		Indexes: info.IndexNames(),
	})
}

func (impl *_Impl) toddl(modelinfo *sqlx.ModelInfo) (*_TableInfo, error) {
	table := &_TableInfo{
		Name: fmt.Sprintf("%s%s", impl.tablename_prefix, strings.ToLower(modelinfo.GoType.Name())),
	}

	ddlbuf := bytes.NewBuffer(nil)
	ddlbuf.WriteString("create table if not exists ")
	ddlbuf.WriteString(table.Name)
	ddlbuf.WriteString(" (\n")

	pks := []string{}
	incrpk := false

	for _, fv := range modelinfo.Fields {
		ddlbuf.WriteByte('\t')
		ddlbuf.WriteString(fv.Name)

		sqltype := fv.SqlType()
		if sqltype == "" {
			var sqltype_err error
			sqltype, sqltype_err = tosqltype(fv.GoType)
			if sqltype_err != nil {
				return nil, sqltype_err
			}
		}
		ddlbuf.WriteString(" ")
		ddlbuf.WriteString(sqltype)

		if fv.IsPrimaryKey() {
			if incrpk {
				return nil, fmt.Errorf("sqlx.dialects.sqlite: table `%s` must have only one primary key if it has autoincrement", table.Name)
			}
			if fv.AutoIncrement() {
				if sqltype != "integer" {
					return nil, fmt.Errorf("sqlx.dialects.sqlite: table `%s` primary key must be integer type", table.Name)
				}
				incrpk = true
				ddlbuf.WriteString(" primary key autoincrement")
			} else {
				pks = append(pks, fv.Name)
			}
		}

		if fv.Nullable() {
		} else {
			ddlbuf.WriteString(" not null")
		}

		if fv.Unique() {
			ddlbuf.WriteString(" unique")
		}

		if expr := fv.CheckExpr(); expr.Valid {
			ddlbuf.WriteString(" check ")
			ddlbuf.WriteString(expr.V)
		}

		if expr := fv.DefaultExpr(); expr.Valid {
			ddlbuf.WriteString(" default ")
			ddlbuf.WriteString(expr.V)
		}

		ddlbuf.WriteString(",\n")
	}

	if incrpk {
		ddlbuf.Truncate(ddlbuf.Len() - 2)
		ddlbuf.WriteString("\n")
	} else {
		if len(pks) < 1 {
			return nil, fmt.Errorf("sqlx.dialects.sqlite: table `%s` must have at least one primary key", table.Name)
		}
		ddlbuf.WriteString("\tprimary key ( ")
		ddlbuf.WriteString(strings.Join(pks, ", "))
		ddlbuf.WriteString(" )\n")
	}
	ddlbuf.WriteString(");")
	table.DDL = ddlbuf.String()

	indexes, err := modelinfo.Indexes()
	if err != nil {
		return nil, err
	}

	for _, item := range indexes {
		if len(item.Name) < 10 {
			item.Name = fmt.Sprintf("%s_of_%s", item.Name, table.Name)
		}

		ddlbuf.Reset()
		ddlbuf.WriteString("create ")
		if item.Unique {
			ddlbuf.WriteString("unique ")
		}
		ddlbuf.WriteString("index if not exists ")
		ddlbuf.WriteString(item.Name)
		ddlbuf.WriteString(" on ")
		ddlbuf.WriteString(table.Name)
		ddlbuf.WriteString(" ( ")

		last := len(item.Fields) - 1
		for i, ele := range item.Fields {
			ddlbuf.WriteString(ele.Name)
			if ele.Desc {
				ddlbuf.WriteString(" desc")
			}
			if i < last {
				ddlbuf.WriteString(", ")
			}
		}
		ddlbuf.WriteString(" );")

		table.Indexes = append(table.Indexes, &_IndexInfo{
			Name: item.Name,
			DDL:  ddlbuf.String(),
		})
	}
	return table, nil
}

var (
	timeType = reflect.TypeFor[time.Time]()
)

func tosqltype(typ reflect.Type) (string, error) {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		{
			return "integer", nil
		}
	case reflect.Float32, reflect.Float64:
		{
			return "real", nil
		}
	case reflect.String:
		{
			return "text", nil
		}
	case reflect.Slice:
		{
			if typ.Elem().Kind() == reflect.Uint8 {
				return "blob", nil
			}
			return "", fmt.Errorf("sqlx.dialects.sqlite: slice type must be []byte, got `%s`", typ.String())
		}
	case reflect.Pointer:
		{
			return tosqltype(typ.Elem())
		}
	case reflect.Struct:
		{
			if _t, ok := sqlx.UnwrapSqlNullType(typ); ok {
				return tosqltype(_t)
			}
			if typ.ConvertibleTo(timeType) {
				return "text", nil
			}
		}
	}
	return "", fmt.Errorf("sqlx.dialects.sqlite: unknown type `%s`, please use `type=xxx` in your struct tag", typ.String())
}
