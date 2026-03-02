package sqlx

import "database/sql"

func (f *ModelField) SqlType() string {
	return f.Opts.Get("type")
}

var (
	defaultpkkeys    = []string{"pk", "primarykey", "primary_key", "primary"}
	defaultincrkeys  = []string{"incr", "autoincrement", "autoincr", "auto_incr", "auto_increment"}
	defaultnullkeys  = []string{"nullable"}
	defaultuniqkeys  = []string{"unique"}
	defaultdefkeys   = []string{"default"}
	defaultcheckkeys = []string{"check"}
	defaultindexkeys = []string{"index", "idx"}
)

func _keys(a []string, b []string) []string {
	if len(a) > 0 {
		return a
	}
	return b
}

func (f *ModelField) HasAny(keys ...string) bool {
	return f.Opts.HasAny(keys...)
}

func (f *ModelField) IsPrimaryKey(keys ...string) bool {
	return f.HasAny(_keys(keys, defaultpkkeys)...)
}

func (f *ModelField) AutoIncrement(keys ...string) bool {
	return f.HasAny(_keys(keys, defaultincrkeys)...)
}

func (f *ModelField) Nullable(keys ...string) bool {
	return f.HasAny(_keys(keys, defaultnullkeys)...)
}

func (f *ModelField) Unique(keys ...string) bool {
	return f.HasAny(_keys(keys, defaultuniqkeys)...)
}

func (f *ModelField) GetAny(keys ...string) sql.Null[string] {
	val, ok := f.Opts.GetAny(keys...)
	if !ok {
		return sql.Null[string]{}
	}
	return sql.Null[string]{V: val, Valid: true}
}

func (f *ModelField) DefaultExpr(keys ...string) sql.Null[string] {
	return f.GetAny(_keys(keys, defaultdefkeys)...)
}

func (f *ModelField) CheckExpr(keys ...string) sql.Null[string] {
	return f.GetAny(_keys(keys, defaultcheckkeys)...)
}
