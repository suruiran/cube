package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/suruiran/cube/sqlx"
)

type IndexNames []string

func (i IndexNames) Value() (driver.Value, error) {
	if len(i) == 0 {
		return nil, nil
	}
	return json.Marshal(i)
}

func (i *IndexNames) Scan(src any) error {
	if src == nil {
		return nil
	}
	switch tv := src.(type) {
	case string:
		{
			return json.Unmarshal([]byte(tv), i)
		}
	case []byte:
		{
			return json.Unmarshal(tv, i)
		}
	default:
		{
			return fmt.Errorf("sqlx: IndexNames Scan: unknown type %T", tv)
		}
	}
}

var (
	_ sql.Scanner   = (*IndexNames)(nil)
	_ driver.Valuer = (*IndexNames)(nil)
)

type _ModelLog struct {
	Name    string     `db:"name" args:"name"`
	Version string     `db:"version" args:"version"`
	Indexes IndexNames `db:"indexes" args:"indexes"`
}

var (
	modelLogDDL = `
	create table if not exists sqlx_model_log (
		name text primary key not null,
		version text not null,
		indexes text
	)
`

	stmtgroup         = sqlx.NewLazyStmtGroup()
	queryAllTableStmt = stmtgroup.New(`select name from sqlite_master where type='table'`)
	queryAllIndexStmt = stmtgroup.New(`select name from sqlite_master where type='index' and tbl_name=${name}`)

	queryAllModelStmt = stmtgroup.New(`select * from sqlx_model_log`)

	deleteOneModelStmt = stmtgroup.New(
		`delete from sqlx_model_log where name = ${name}`,
	)

	insertModelLogStmt = stmtgroup.New(
		`insert into sqlx_model_log (name, version, indexes) values (${name}, ${version}, ${indexes})`,
	)
)

func initModelLogs(ctx context.Context) error {
	exec := sqlx.MustExecutor(ctx)
	if _, err := exec.ExecContext(
		ctx,
		modelLogDDL,
	); err != nil {
		return err
	}
	return stmtgroup.InitAll(ctx, nil)
}

func (i *_Impl) queryAllModels(ctx context.Context) ([]_ModelLog, error) {
	models, err := sqlx.All[_ModelLog](ctx, queryAllModelStmt.Must(), 100)
	if err != nil {
		return nil, err
	}
	if !i.syncPrevs {
		return models, nil
	}

	type NameItem struct {
		Name string `db:"name" args:"name"`
	}

	umodels := sqlx.NewUniqueSlice(func(t *_ModelLog) string { return t.Name })
	umodels.V = models

	mapop := sqlx.NewMap(queryAllTableStmt.Must(), func(ctx context.Context, t *NameItem) (string, error) { return t.Name, nil })
	for tblname, err := range mapop.Iter(ctx) {
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(tblname, i.tablename_prefix) {
			continue
		}
		if umodels.Has(tblname) {
			continue
		}
		if tblname == "sqlx_model_log" {
			continue
		}

		item := _ModelLog{
			Name: tblname,
		}

		indexNames, err := sqlx.All[NameItem](ctx, queryAllIndexStmt.Must(), 20, NameItem{Name: tblname})
		if err != nil {
			return nil, err
		}
		for _, idxname := range indexNames {
			if strings.HasPrefix(idxname.Name, "sqlite_") {
				continue
			}
			item.Indexes = append(item.Indexes, idxname.Name)
		}

		umodels.Push(item)
	}
	return umodels.V, nil
}

func dropModelLog(ctx context.Context, log *_ModelLog) (string, error) {
	exec := sqlx.MustExecutor(ctx)
	for _, index := range log.Indexes {
		if _, err := exec.ExecContext(
			ctx,
			fmt.Sprintf(`drop index if exists %s`, index),
		); err != nil {
			return "", err
		}
	}

	archived_name := fmt.Sprintf(`zzZZZ_%s_v%s`, log.Name, log.Version)
	if _, err := exec.ExecContext(
		ctx,
		fmt.Sprintf(`alter table %s rename to %s`, log.Name, archived_name),
	); err != nil {
		return "", err
	}
	_, err := deleteOneModelStmt.Exec(ctx, log)
	return archived_name, err
}

func insertModelLog(ctx context.Context, log *_ModelLog) error {
	if IsDryRun(ctx) {
		slog.InfoContext(ctx, "InsertModelLog", "name", log.Name, "version", log.Version, "indexes", log.Indexes)
		return nil
	}
	_, err := insertModelLogStmt.ExecContext(ctx, log)
	return err
}
