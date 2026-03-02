package sqlx

import (
	"context"
	"log/slog"
	"testing"
)

type _FakeDialect struct{}

func (*_FakeDialect) EnsureTables(ctx context.Context, infos []*ModelInfo) error {
	return nil
}

func (*_FakeDialect) ParamPlaceholder(index int) string {
	return "?"
}

var _ IDialect = (*_FakeDialect)(nil)

func TestStmtPrepare(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	ctx := WithDialect(t.Context(), &_FakeDialect{})

	_, err := NewStmt(ctx, "SELECT * FROM table WHERE id = ${id}")
	if err != nil {
		t.Fatal(err)
	}
}
