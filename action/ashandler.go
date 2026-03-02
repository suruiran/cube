package action

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suruiran/cube/logx"
	"go.uber.org/ratelimit"
)

var (
	adminrl = ratelimit.New(1)
)

type ISession interface {
	Take()
	TakeN(times int)
}

type ISessionProvider interface {
	Get(req *http.Request) (ISession, bool)
	Ensure(req *http.Request, respw http.ResponseWriter) ISession
}

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

	handler := http.HandlerFunc(func(respw http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(respw, req.Body, MaxRequestBodySize)
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

		logic_handled := false

		defer func() {
			rv := recover()
			if rv == nil {
				return
			}
			if !logic_handled {
				he, ok := rv.(IHttpError)
				if ok {
					respw.WriteHeader(he.Code())
					_, _ = respw.Write([]byte(he.Error()))
					return
				}
				respw.WriteHeader(http.StatusInternalServerError)
			}
			logger.Error("ActionHandlePaniced", slog.String("name", actionname), logx.PanicWithStacktrace(rv, nil))
		}()

		var session ISession
		var err error

		switch actionitem.Opts.SessionPolicy {
		case SessionPolicyKindAuto:
			{
				session = group.sessionprovider.Ensure(req, respw)
				break
			}
		case SessionPolicyKindRequire:
			{
				var ok bool
				session, ok = group.sessionprovider.Get(req)
				if !ok {
					if cfg.Debug {
						session = group.sessionprovider.Ensure(req, respw)
					} else {
						respw.WriteHeader(http.StatusBadRequest)
						_, _ = respw.Write([]byte("Session Is Required."))
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

		if actionitem.Opts.RequireAdmin {
			parts := strings.Split(req.Header.Get("X-Admin-Auth"), "/")
			if len(parts) != 2 {
				respw.WriteHeader(http.StatusNotFound)
				return
			}

			adminrl.Take()

			filename, code := parts[0], parts[1]
			filename = filepath.Base(filepath.Clean(filename))
			fp := filepath.Join(cfg.AdminAuthDir, filename)
			buf, rerr := os.ReadFile(fp)
			if rerr != nil {
				respw.WriteHeader(http.StatusNotFound)
				return
			}
			_ = os.Remove(fp)

			if string(buf) != code {
				respw.WriteHeader(http.StatusNotFound)
				return
			}
		} else {
			remoteaddr := group.remoteipprovider(req)
			if remoteaddr == "" {
				netremote := req.RemoteAddr
				if strings.HasPrefix(netremote, "10.0.0.") || strings.HasPrefix(netremote, "127.0.0.1") {
				} else {
					logger.Error("Action: Direct Call", slog.String("Address", req.RemoteAddr))
					respw.WriteHeader(http.StatusNotFound)
					return
				}
			}
		}

		if session != nil {
			if actionitem.Opts.RateLimitTakeN > 0 {
				session.TakeN(actionitem.Opts.RateLimitTakeN)
			} else {
				session.Take()
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

		logic_handled = true
		if err == sql.ErrNoRows {
			respw.WriteHeader(http.StatusNotFound)
			return
		}
		he, ok := err.(IHttpError)
		if ok {
			respw.WriteHeader(he.Code())
			_, _ = respw.Write([]byte(he.Error()))
			return
		}
		respw.WriteHeader(http.StatusInternalServerError)
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
