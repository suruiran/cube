package sqlx

import (
	"log/slog"
	"sync"
)

var (
	_logger     *slog.Logger
	_loggerinit sync.Once
)

func Logger() *slog.Logger {
	_loggerinit.Do(func() {
		_logger = slog.Default().WithGroup("sqlx")
	})
	return _logger
}
