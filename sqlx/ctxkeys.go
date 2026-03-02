package sqlx

type _CtxKey int

const (
	_CtxKeyTx _CtxKey = iota
	_CtxKeyDB
	_CtxKeyExec
	_CtxKeyDialect
)
