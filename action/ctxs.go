package action

import (
	"context"
	"errors"
	"net/http"
)

type _CtxKey int

const (
	_CtxKeyForSession _CtxKey = iota
	_CtxKeyForRequest
)

func PeekRequest(ctx context.Context) (*http.Request, bool) {
	av := ctx.Value(_CtxKeyForRequest)
	if av == nil {
		return nil, false
	}
	req, ok := av.(*http.Request)
	return req, ok
}

var (
	errEmptyRequestCtx = errors.New("ctx: empty request")
)

func MustPeekRequest(ctx context.Context) *http.Request {
	v, ok := PeekRequest(ctx)
	if !ok {
		panic(errEmptyRequestCtx)
	}
	return v
}

func PeekSession(ctx context.Context) (ISession, bool) {
	av := ctx.Value(_CtxKeyForSession)
	if av == nil {
		return nil, false
	}
	session, ok := av.(ISession)
	return session, ok
}
