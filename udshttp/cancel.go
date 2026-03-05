package udshttp

import "context"

type _CancelAction struct {
	ReqId int64
}

type _CancelActionResult struct{}

type _CtxType int

const (
	_CtxKeyForReqId _CtxType = iota
)

func PeekReqId(ctx context.Context) int64 {
	va := ctx.Value(_CtxKeyForReqId)
	if va == nil {
		return 0
	}
	return va.(int64)
}

func Cancel(reqid int64) {
	fv, ok := cancelbyreqids.LoadAndDelete(reqid)
	if ok {
		fn := fv.(func())
		fn()
	}
}
