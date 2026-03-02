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

func PeekRequestFromCtx(ctx context.Context) (*http.Request, bool) {
	av := ctx.Value(_CtxKeyForRequest)
	if av == nil {
		return nil, false
	}
	req, ok := av.(*http.Request)
	return req, ok
}

var (
	errEmptyRequestCtx = errors.New("cube.action: empty request")
)

func MustPeekRequestFromCtx(ctx context.Context) *http.Request {
	v, ok := PeekRequestFromCtx(ctx)
	if !ok {
		panic(errEmptyRequestCtx)
	}
	return v
}

func PeekSessionFromCtx(ctx context.Context) (ISession, bool) {
	av := ctx.Value(_CtxKeyForSession)
	if av == nil {
		return nil, false
	}
	session, ok := av.(ISession)
	return session, ok
}
