package sqlx

import (
	"log/slog"
	"time"
)

type _StmtLogItem struct {
	stmt  *Stmt
	args  []any
	begin time.Time
}

func (item _StmtLogItem) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("sql", item.stmt.rawsql),
		slog.Duration("duration", time.Since(item.begin)),
	}
	for idx, arg := range item.args {
		attrs = append(attrs, slog.Any(item.stmt.lognames[idx], arg))
	}
	return slog.GroupValue(attrs...)
}

var _ slog.LogValuer = _StmtLogItem{}
