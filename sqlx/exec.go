package sqlx

import (
	"context"
	"database/sql"
	"fmt"
)

type IExecutor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, _CtxKeyTx, tx)
}

func WithDB(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, _CtxKeyDB, db)
}

func WithDialect(ctx context.Context, dialect IDialect) context.Context {
	return context.WithValue(ctx, _CtxKeyDialect, dialect)
}

func WithExec(ctx context.Context, exec IExecutor) context.Context {
	return context.WithValue(ctx, _CtxKeyExec, exec)
}

func MustTx(ctx context.Context) *sql.Tx {
	return ctx.Value(_CtxKeyTx).(*sql.Tx)
}

func PeekTx(ctx context.Context) *sql.Tx {
	tx := ctx.Value(_CtxKeyTx)
	if tx != nil {
		return tx.(*sql.Tx)
	}
	return nil
}

func MustDB(ctx context.Context) *sql.DB {
	return ctx.Value(_CtxKeyDB).(*sql.DB)
}

func PeekDB(ctx context.Context) *sql.DB {
	db := ctx.Value(_CtxKeyDB)
	if db == nil {
		return nil
	}
	return db.(*sql.DB)
}

func MustDialect(ctx context.Context) IDialect {
	return ctx.Value(_CtxKeyDialect).(IDialect)
}

func PeekDialect(ctx context.Context) IDialect {
	dialect := ctx.Value(_CtxKeyDialect)
	if dialect == nil {
		return nil
	}
	return dialect.(IDialect)
}

func MustExecutor(ctx context.Context) IExecutor {
	_exec := ctx.Value(_CtxKeyExec)
	if _exec != nil {
		return _exec.(IExecutor)
	}
	tx := PeekTx(ctx)
	if tx != nil {
		return tx
	}
	return MustDB(ctx)
}

func PeekExecutor(ctx context.Context) IExecutor {
	_exec := ctx.Value(_CtxKeyExec)
	if _exec != nil {
		return _exec.(IExecutor)
	}

	tx := PeekTx(ctx)
	if tx != nil {
		return tx
	}
	return PeekDB(ctx)
}

func TxScope(ctx context.Context, f func(ctx context.Context) error, opts *sql.TxOptions) error {
	if _tv := ctx.Value(_CtxKeyTx); _tv != nil {
		return fmt.Errorf("cube.sqlx: already in transaction")
	}
	tx, err := MustDB(ctx).BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	ctx = WithTx(ctx, tx)
	if err := f(ctx); err != nil {
		return err
	}
	return tx.Commit()
}
