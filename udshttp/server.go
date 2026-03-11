package udshttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/suruiran/cube"
	"github.com/suruiran/cube/logx"
)

type Server struct {
	uuid     uuid.UUID
	fp       string
	mux      *http.ServeMux
	listener net.Listener
	logger   *slog.Logger
}

var (
	ErrServerIsRunning = errors.New("running")
)

func NewServer(fp string, mux *http.ServeMux) (*Server, error) {
	return NewServerWithLogger(fp, mux, nil)
}

func NewServerWithLogger(fp string, mux *http.ServeMux, log *slog.Logger) (*Server, error) {
	if _IsRunning(fp) {
		return nil, ErrServerIsRunning
	}
	CleanFiles(fp)

	listener, err := net.Listen("unix", fp)
	if err != nil {
		return nil, err
	}
	writepid(fp)

	cube.OnDeath(func(wg *sync.WaitGroup) {
		CleanFiles(fp)
	})

	if log == nil {
		if cube.Env("CUBE_UDS_LOG_DEFAULT", false) {
			log = slog.Default()
		} else {
			nobuf := cube.Env("CUBE_UDS_LOG_NOBUFFERED", false)
			bufsize := 4096
			if nobuf {
				bufsize = -1
			}

			log, err = logx.New(&logx.Opts{
				Filename:   fmt.Sprintf("%s.log", fp),
				Level:      cube.Env("CUBE_UDS_LOG_LEVEL", slog.LevelInfo),
				AddSource:  cube.Env("CUBE_UDS_LOG_SOURCE", false),
				WithStdout: cube.Env("CUBE_UDS_LOG_STDOUT", false),
				BufferSize: bufsize,
				Rolling: &logx.RollingOptions{
					Kind:    logx.RollingKindDaily,
					Backups: 30,
				},
			})
			if err != nil {
				return nil, err
			}
		}
	}

	serv := &Server{
		uuid:     uuid.New(),
		fp:       fp,
		listener: listener,
		mux:      mux,
		logger:   log,
	}

	Register(serv.mux, serv.logger, func(ctx context.Context, input _CancelAction) (_CancelActionResult, error) {
		Cancel(input.ReqId)
		return _CancelActionResult{}, nil
	})
	return serv, nil
}

var (
	HeaderServerUUID   = "Cube-Server-UUID"
	HeaderIsCancelable = "Cube-Is-Cancelable"
	HeaderReqId        = "Cube-Req-ID"
)

func (serv *Server) Run() error {
	serv.logger.Info("udshttp server start", slog.String("fp", serv.fp), slog.Int("pid", os.Getpid()))
	return http.Serve(serv.listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderServerUUID, serv.uuid.String())
		serv.mux.ServeHTTP(w, r)
	}))
}

var (
	cancelbyreqids sync.Map
	UserAgent      = "cube.udshttp"
	OnReq          func(ctx context.Context, r *http.Request) bool
)

func Register[Input any, Output any](mux *http.ServeMux, logger *slog.Logger, fn func(ctx context.Context, input Input) (Output, error)) {
	it := reflect.TypeFor[Input]()
	isptr := false
	if it.Kind() == reflect.Pointer {
		it = it.Elem()
		isptr = true
	}

	if it.Kind() != reflect.Struct {
		panic(fmt.Errorf("udshttp: input is not a struct/*struct"))
	}
	ot := reflect.TypeFor[Output]()
	if ot.Kind() == reflect.Pointer {
		ot = ot.Elem()
	}
	if ot.Kind() != reflect.Struct {
		panic(fmt.Errorf("udshttp: output is not a struct/*struct"))
	}

	mux.HandleFunc(fmt.Sprintf("/%s", strings.ToLower(it.Name())), func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			ev := recover()
			if ev != nil {
				logger.Error("udshttp server handle paniced", slog.String("path", r.URL.Path), slog.Any("err", ev))
			}
		}()

		if OnReq != nil && !OnReq(r.Context(), r) {
			w.WriteHeader(http.StatusForbidden)
			logger.Error("rejected by OnReq", slog.String("path", r.URL.Path))
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			logger.Error("udshttp server handle method not allowed", slog.String("path", r.URL.Path))
			return
		}
		if r.Header.Get("User-Agent") != UserAgent {
			w.WriteHeader(http.StatusForbidden)
			logger.Error("udshttp server handle user agent not allowed", slog.String("path", r.URL.Path), slog.String("user-agent", r.Header.Get("User-Agent")))
			return
		}
		pid := r.Header.Get("Os-Pid")
		if pid == "" {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("udshttp server handle os pid not provided", slog.String("path", r.URL.Path))
			return
		}

		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("udshttp server handle read body failed", slog.String("path", r.URL.Path), slog.Any("err", err))
			return
		}

		var input Input
		if isptr {
			ptrv := reflect.New(it)
			err = cube.UnmarshalJSON(bs, ptrv.Interface())
			if err == nil {
				input = ptrv.Interface().(Input)
			}
		} else {
			err = cube.UnmarshalJSON(bs, &input)
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logger.Error("udshttp server handle unmarshal body failed", slog.String("path", r.URL.Path), slog.Any("err", err))
			return
		}

		is_cancelable := r.Header.Get(HeaderIsCancelable) == "true"
		reqid, _ := strconv.ParseInt(r.Header.Get(HeaderReqId), 10, 64)

		ctx, cancel := context.WithCancel(r.Context())
		cancelbyreqids.Store(reqid, func() {
			cancel()
			cancelbyreqids.Delete(reqid)
		})
		if is_cancelable {
			ctx = context.WithValue(context.Background(), _CtxKeyForReqId, reqid)
		} else {
			defer cancel()
		}

		output, err := fn(ctx, input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			logger.Error("udshttp server handle function failed", slog.String("path", r.URL.Path), slog.Any("err", err), slog.Any("input", input))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Accept", "application/json")
		bs = cube.MustMarshalJSON(output)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bs)

		logger.Debug("udshttp server handle success", slog.String("path", r.URL.Path), slog.Any("input", input), slog.Any("output", output))
	})
}

func (serv *Server) Close() {
	if serv.listener != nil {
		serv.listener.Close() //nolint:errcheck
	}
	CleanFiles(serv.fp)
}

func (serv *Server) Logger() *slog.Logger {
	return serv.logger
}
