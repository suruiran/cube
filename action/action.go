package action

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
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
	ErrorMap     func(err error) (IHttpError, bool)
}

type IAdminChecker interface {
	Check(ctx context.Context, ip string, req *http.Request) error
	Do(ctx context.Context, cli *http.Client, req *http.Request) (*http.Response, error)
}

type ActionGroup struct {
	sessionprovider  ISessionProvider
	remoteipprovider IRemoteIPProvider
	adminchecker     IAdminChecker
	cfg              *Config
	logger           *slog.Logger
	module_prefix    string
	apiactions       map[string]_ActItem
	//TODO auto generate api documantation and typescript types.
	//TODO a validate impl base on rbc
	iotypes map[string]_IoTypes
	optsfnc func(opts *ActionOptions) *ActionOptions
}

func NewGroup(module_prefix string, optsfnc func(opts *ActionOptions) *ActionOptions) *ActionGroup {
	return &ActionGroup{
		module_prefix: module_prefix,
		iotypes:       map[string]_IoTypes{},
		apiactions:    map[string]_ActItem{},
		optsfnc:       optsfnc,
	}
}

func (group *ActionGroup) WithSessionProvider(sessionprovider ISessionProvider) *ActionGroup {
	group.sessionprovider = sessionprovider
	return group
}

func (group *ActionGroup) WithLogger(logger *slog.Logger) *ActionGroup {
	group.logger = logger
	return group
}

func (group *ActionGroup) WithConfig(cfg *Config) *ActionGroup {
	group.cfg = cfg
	return group
}

func (group *ActionGroup) WithRemoteIpProvider(remoteipprovider IRemoteIPProvider) *ActionGroup {
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

// ISerializer
// impls not need to close req.Body in Deserialize method
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
	name := group.nameof(fnc)

	inputtype := reflect.TypeFor[Input]()
	outputtype := reflect.TypeFor[Output]()
	if inputtype.Kind() != reflect.Struct || outputtype.Kind() != reflect.Pointer {
		panic(fmt.Errorf("action: bad input(should be a struct) or output type(should be a pointer to a struct), %s", name))
	}

	isreq := inputtype.AssignableTo(httpRequestType) && inputtype.Size() == httpRequestType.Size()
	isbindable := reflect.PointerTo(inputtype).Implements(iBindFroHttpType)

	outputtype = outputtype.Elem()
	if outputtype.Kind() != reflect.Struct {
		panic(fmt.Errorf("action: bad input(should be a struct) or output type(should be a pointer to a struct), %s", name))
	}

	group.addapi(
		name,
		func(ctx context.Context, req *http.Request, respw http.ResponseWriter) error {
			var input Input
			var inputptr = &input
			if isreq {
				inputptr = (*Input)(unsafe.Pointer(req))
			} else {
				if isbindable {
					var bind = (any)(&input).(IBind)
					if err := bind.Bind(req); err != nil {
						return err
					}
				} else {
					if err := serializer.Deserialize(req, &input); err != nil {
						return NewHttpError(http.StatusBadRequest, "empty request body")
					}
				}
			}

			out, err := fnc(ctx, inputptr)
			if err != nil {
				return err
			}

			sc := out.Code()
			maps.Copy(respw.Header(), out.Headers())
			maps.Copy(respw.Header(), serializer.Header())
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
	group.iotypes[name] = _IoTypes{in: inputtype, out: outputtype}
}
