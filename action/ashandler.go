package action

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/suruiran/cube/logx"
)

type ISession interface {
	Take() bool
	TakeN(times int) bool
}

type ISessionProvider interface {
	Get(req *http.Request) (ISession, bool)
	Ensure(req *http.Request, respw http.ResponseWriter) (ISession, error)
}

var (
	internalErrorMsg = []byte("internal server error")
)

type _Respw struct {
	http.ResponseWriter
	wrote bool
}

func (rw *_Respw) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *_Respw) Write(b []byte) (int, error) {
	rw.wrote = true
	return rw.ResponseWriter.Write(b)
}

func (rw *_Respw) WriteHeader(statusCode int) {
	rw.wrote = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

var _ http.ResponseWriter = (*_Respw)(nil)

func (group *ActionGroup) ToHandler(actiongetter func(req *http.Request) string) http.Handler {
	logger := group.logger
	cfg := group.cfg

	if logger == nil || cfg == nil || group.sessionprovider == nil || group.remoteipprovider == nil {
		panic(errors.New("action: nil component, logger/cfg/sessionprovider/remoteipprovider"))
	}
	actions := slices.Collect(maps.Keys(group.apiactions))
	for idx, name := range actions {
		info := group.apiactions[name]
		if info.Opts != nil {
			if !cfg.Debug && info.Opts.OnlyDebug {
				continue
			}
			if info.Opts.RequireAdmin {
				actions[idx] = fmt.Sprintf("Admin:(%s)", actions[idx])
			}
			if info.Opts.OnlyDebug {
				actions[idx] = fmt.Sprintf("Dev:(%s)", actions[idx])
			}
		}
	}
	slices.Sort(actions)
	logger.Info("ActionGroup", slog.String("module prefix", group.module_prefix), slog.String("actions", strings.Join(actions, ", ")))

	var tohe func(err error) (IHttpError, bool)
	if cfg != nil && cfg.ErrorMap != nil {
		tohe = func(err error) (IHttpError, bool) {
			if _, ok := err.(IHttpError); ok {
				return err.(IHttpError), true
			}
			return cfg.ErrorMap(err)
		}
	} else {
		tohe = func(err error) (IHttpError, bool) {
			if _, ok := err.(IHttpError); ok {
				return err.(IHttpError), true
			}
			return nil, false
		}
	}

	handler := http.HandlerFunc(func(respw http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(respw, req.Body, MaxRequestBodySize)
		_w := &_Respw{ResponseWriter: respw}
		respw = _w

		if req.Method == http.MethodOptions || req.Method == http.MethodConnect || req.Method == http.MethodTrace {
			respw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		actionname := actiongetter(req)
		if actionname == "" {
			respw.WriteHeader(http.StatusNotFound)
			return
		}
		actionitem, ok := group.apiactions[actionname]
		if !ok {
			respw.WriteHeader(http.StatusNotFound)
			return
		}

		if actionitem.Opts.OnlyDebug {
			if !cfg.Debug {
				respw.WriteHeader(http.StatusNotFound)
				return
			}
		}

		senderr := func(err error) {
			if _w.wrote {
				return
			}
			he, ok := tohe(err)
			if !ok {
				respw.WriteHeader(http.StatusInternalServerError)
				return
			}
			respw.WriteHeader(he.Code())
			_, _ = respw.Write([]byte(he.Error()))
		}

		defer func() {
			_ = req.Body.Close()

			rv := recover()
			if rv == nil {
				return
			}
			if !_w.wrote {
				he, ok := rv.(IHttpError)
				if ok {
					code := he.Code()
					respw.WriteHeader(code)
					if code >= 500 {
						_, _ = respw.Write(internalErrorMsg)
					} else {
						_, _ = respw.Write([]byte(he.Error()))
					}
					return
				}
				respw.WriteHeader(http.StatusInternalServerError)
			}
			logger.Error("ActionHandlePaniced", slog.String("name", actionname), logx.RecoveredWithStacktrace(rv, nil), slog.Bool("wrote", _w.wrote))
		}()

		var session ISession
		var err error
		switch actionitem.Opts.SessionPolicy {
		case SessionPolicyKindAuto:
			{
				session, err = group.sessionprovider.Ensure(req, respw)
				break
			}
		case SessionPolicyKindRequire:
			{
				var ok bool
				session, ok = group.sessionprovider.Get(req)
				if !ok {
					if cfg.Debug {
						session, err = group.sessionprovider.Ensure(req, respw)
					} else {
						respw.WriteHeader(http.StatusForbidden)
						return
					}
				}
				break
			}
		case SessionPolicyKindNone:
			{
				break
			}
		}
		if err != nil {
			senderr(err)
			return
		}

		remoteaddr := group.remoteipprovider.Get(req)
		if actionitem.Opts.RequireAdmin {
			if group.adminchecker == nil {
				respw.WriteHeader(http.StatusNotFound)
				return
			}
			if err := group.adminchecker.Check(req.Context(), remoteaddr, req); err != nil {
				respw.WriteHeader(http.StatusNotFound)
				group.logger.Error("RejectedAdminCall.Check", slog.String("remoteaddr", remoteaddr), logx.ErrorWithStacktrace(err, nil))
				return
			}
		}

		if session != nil {
			rlok := false
			if actionitem.Opts.RateLimitTakeN > 0 {
				rlok = session.TakeN(actionitem.Opts.RateLimitTakeN)
			} else {
				rlok = session.Take()
			}
			if !rlok {
				respw.WriteHeader(http.StatusTooManyRequests)
				return
			}
			if req.Context().Err() != nil {
				return
			}
		}

		ctx := context.WithValue(req.Context(), _CtxKeyForSession, session)
		ctx = context.WithValue(ctx, _CtxKeyForRequest, req)

		err = actionitem.Fnc(ctx, req, respw)
		if err == nil {
			return
		}

		senderr(err)
		logger.Error("ActionHandleFailed", slog.String("name", actionname), logx.ErrorWithStacktrace(err, nil))
	})

	if cfg.Debug {
		tmp := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "*")
			w.Header().Add("Access-Control-Allow-Headers", "*")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			tmp(w, r)
		})
	}
	return handler
}
