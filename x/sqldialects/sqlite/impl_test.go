package sqlite

import (
	"database/sql"
	"log/slog"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/suruiran/cube/sqlx"
)

type User struct {
	ID   int    `sql:"id,pk;incr" db:"id"`
	Name string `sql:"name" db:"name"`
	Age  int    `sql:"age" db:"age"`
}

func init() {
	sqlx.DisableEnsureBackedUp()
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func TestEnsureTable(t *testing.T) {
	db, err := sql.Open("sqlite3", "./t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() //nolint:errcheck

	ctx := sqlx.WithDB(t.Context(), db)
	ctx = sqlx.WithDialect(
		ctx,
		New(
			&Options{
				SyncPrevs: true,
			},
		),
	)

	sqlx.AsModel[User]()

	if err := sqlx.Sync(ctx); err != nil {
		t.Fatal(err)
	}
}
