package action

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"unsafe"

	"github.com/suruiran/cube"
)

type ApiAction func(ctx context.Context, req *http.Request, respw http.ResponseWriter) error

type _IoTypes struct {
	in  reflect.Type
	out reflect.Type
}

type SessionPolicyKind int

const (
	SessionPolicyKindAuto SessionPolicyKind = iota
	SessionPolicyKindRequire
	SessionPolicyKindNone
)

type ActionOptions struct {
	SessionPolicy  SessionPolicyKind
	RateLimitTakeN int
	RequireAdmin   bool
	OnlyDebug      bool
}

type _ActItem struct {
	Fnc  ApiAction
	Opts *ActionOptions
}

type Config struct {
	Debug        bool
	AdminAuthDir string
}

type ActionGroup struct {
	sessionprovider  ISessionProvider
	remoteipprovider func(req *http.Request) string
	cfg              *Config
	logger           *slog.Logger
	module_prefix    string
	apiactions       map[string]_ActItem
	iotypes          map[string]_IoTypes
	optsfnc          func(opts *ActionOptions) *ActionOptions
}

func NewGroup(module_prefix string, optsfnc func(opts *ActionOptions) *ActionOptions) *ActionGroup {
	return &ActionGroup{
		module_prefix: module_prefix,
		iotypes:       map[string]_IoTypes{},
		apiactions:    map[string]_ActItem{},
		optsfnc:       optsfnc,
	}
}

func (group *ActionGroup) SetSessionProvider(sessionprovider ISessionProvider) *ActionGroup {
	group.sessionprovider = sessionprovider
	return group
}

func (group *ActionGroup) SetLogger(logger *slog.Logger) *ActionGroup {
	group.logger = logger
	return group
}

func (group *ActionGroup) SetConfig(cfg *Config) *ActionGroup {
	group.cfg = cfg
	return group
}

func (group *ActionGroup) SetRemoteIpProvider(remoteipprovider func(req *http.Request) string) *ActionGroup {
	group.remoteipprovider = remoteipprovider
	return group
}

func (group *ActionGroup) nameof(act any) string {
	fncname := cube.FuncName(act)
	actname := strings.ToLower(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ToLower(strings.TrimPrefix(fncname, group.module_prefix)),
			"\\", ".",
		),
		"/", ".",
	))
	return actname
}

const (
	MaxRequestBodySize = 1024_00
)

func (group *ActionGroup) RawApi(fnc ApiAction, opts *ActionOptions) {
	group.addapi(group.nameof(fnc), fnc, opts)
}

func (group *ActionGroup) addapi(name string, fnc ApiAction, opts *ActionOptions) {
	if opts == nil {
		opts = &ActionOptions{}
	}
	_, ok := group.apiactions[name]
	if ok {
		panic(fmt.Errorf("action: %s, is already exists", name))
	}
	if group.optsfnc != nil {
		opts = group.optsfnc(opts)
	}
	group.apiactions[name] = _ActItem{
		Fnc:  fnc,
		Opts: opts,
	}
}

type ISerializer interface {
	Deserialize(req *http.Request, dst any) error
	Header() http.Header
	Serialize(ctx context.Context, w io.Writer, src any) error
}

type IBind interface {
	Bind(req *http.Request) error
}

var (
	iBindFroHttpType = reflect.TypeFor[IBind]()
	httpRequestType  = reflect.TypeFor[http.Request]()
)

func Api[Input any, Output IHttpOutput](
	group *ActionGroup,
	serializer ISerializer,
	fnc func(ctx context.Context, input *Input) (Output, error),
	opts *ActionOptions,
) {
	actionname := group.nameof(fnc)

	inputtype := reflect.TypeFor[Input]()
	outputtype := reflect.TypeFor[Output]()
	if inputtype.Kind() != reflect.Struct || outputtype.Kind() != reflect.Pointer {
		panic(fmt.Errorf("action: bad input or output type"))
	}

	isreq := inputtype.AssignableTo(httpRequestType)
	isbindable := reflect.PointerTo(inputtype).Implements(iBindFroHttpType)
	outputtype = outputtype.Elem()
	if outputtype.Kind() != reflect.Struct {
		panic(fmt.Errorf("action: bad input or output type"))
	}

	group.addapi(
		actionname,
		func(ctx context.Context, req *http.Request, respw http.ResponseWriter) error {
			var input Input
			if isreq {
				input = *(*Input)(unsafe.Pointer(&req))
			} else {
				if isbindable {
					var bind = (any)(&input).(IBind)
					if err := bind.Bind(req); err != nil {
						return err
					}
				} else {
					switch req.Method {
					case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect, http.MethodDelete:
						{
							return NewHttpError(http.StatusMethodNotAllowed, "")
						}
					}
					if err := serializer.Deserialize(req, &input); err != nil {
						return NewHttpError(http.StatusNotAcceptable, "")
					}
				}
			}

			out, err := fnc(ctx, &input)
			if err != nil {
				return err
			}

			sc := out.Code()
			for k, vs := range out.Headers() {
				respw.Header().Del(k)
				for _, v := range vs {
					respw.Header().Add(k, v)
				}
			}
			for k, vs := range serializer.Header() {
				respw.Header().Del(k)
				for _, v := range vs {
					respw.Header().Add(k, v)
				}
			}
			respw.WriteHeader(sc)

			rb, ok := out.BytesBody()
			if ok {
				_, err = respw.Write(rb)
				return err
			}
			return serializer.Serialize(ctx, respw, out)
		},
		opts,
	)
	group.iotypes[actionname] = _IoTypes{in: inputtype, out: outputtype}
}
