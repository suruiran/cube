package action

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
)

type _JsonSerializer struct{}

func (js _JsonSerializer) Deserialize(req *http.Request, dst any) error {
	ct := req.Header.Get("Content-Type")
	if ct == "" {
		return fmt.Errorf("missing Content-Type header")
	}
	ct = strings.ToLower(ct)
	if !strings.HasPrefix(ct, "application/json") {
		return fmt.Errorf("content type is not json")
	}
	return json.NewDecoder(req.Body).Decode(dst)
}

func (js _JsonSerializer) Header() http.Header {
	return http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}
}

func (js _JsonSerializer) Serialize(ctx context.Context, w io.Writer, src any) error {
	return json.NewEncoder(w).Encode(src)
}

var JSONSerializer ISerializer = _JsonSerializer{}

func JSONApi[Input any, Output IHttpOutput](
	group *ActionGroup,
	fnc func(ctx context.Context, input *Input) (Output, error),
	opts *ActionOptions,
) {
	Api(group, JSONSerializer, fnc, opts)
}

type StringMap struct {
	Map map[string]string
}

func (s *StringMap) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.Map)
}

var _ json.Unmarshaler = (*StringMap)(nil)
