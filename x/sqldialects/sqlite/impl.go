package sqlite

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"

	"github.com/suruiran/cube/sqlx"
)

type OnArchivedFunc func(ctx context.Context, current, archived string) error

type _Impl struct {
	tablename_prefix string
	onArchived       OnArchivedFunc
	dryrun           bool
	syncPrevs        bool
}

type Options struct {
	TablenamePrefix string
	DryRun          bool
	SyncPrevs       bool
	OnArchived      OnArchivedFunc
}

func New(opts *Options) sqlx.IDialect {
	if opts == nil {
		opts = &Options{}
	}
	if len(opts.TablenamePrefix) < 1 {
		opts.TablenamePrefix = "sqlx_"
	}
	return &_Impl{
		tablename_prefix: opts.TablenamePrefix,
		onArchived:       opts.OnArchived,
		dryrun:           opts.DryRun,
		syncPrevs:        opts.SyncPrevs,
	}
}

type _CtxKey int

const (
	dryrunKey _CtxKey = iota
)

func IsDryRun(ctx context.Context) bool {
	return ctx.Value(dryrunKey) == true
}

type _FakeExec struct {
	exec sqlx.IExecutor
}

func (fe *_FakeExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	slog.InfoContext(ctx, "ExecContext", "query", query, "args", args)
	return nil, nil
}

func (fe *_FakeExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return fe.exec.QueryContext(ctx, query, args...)
}

var _ sqlx.IExecutor = (*_FakeExec)(nil)

func (i *_Impl) EnsureTables(ctx context.Context, infos []*sqlx.ModelInfo) error {
	if err := initModelLogs(ctx); err != nil {
		return err
	}
	return sqlx.TxScope(
		ctx,
		func(ctx context.Context) error {
			if i.dryrun {
				exec := sqlx.MustExecutor(ctx)
				ctx = sqlx.WithExec(ctx, &_FakeExec{exec: exec})
				ctx = context.WithValue(ctx, dryrunKey, true)
			}

			prevmodels, err := i.queryAllModels(ctx)
			if err != nil {
				return err
			}
			for _, model := range infos {
				table, err := i.toddl(model)
				if err != nil {
					return err
				}
				archived_name := ""
				idx := slices.IndexFunc(prevmodels, func(ele _ModelLog) bool { return ele.Name == table.Name })
				if idx > -1 {
					prev := &prevmodels[idx]
					if prev.Version == table.Version() {
						continue
					}
					archived_name, err = dropModelLog(ctx, prev)
					if err != nil {
						return err
					}
				}

				if err := createTable(ctx, table); err != nil {
					return err
				}
				if archived_name != "" && i.onArchived != nil {
					if err := i.onArchived(ctx, table.Name, archived_name); err != nil {
						return err
					}
				}
			}
			return nil
		},
		nil,
	)
}

func (i *_Impl) ParamPlaceholder(index int) string {
	return "?"
}

var _ sqlx.IDialect = (*_Impl)(nil)
