package action

import (
	"bytes"
	"context"
	"net/http"

	"github.com/suruiran/cube"
)

func NewJsonRequest[T any](ctx context.Context, method string, url string, val T) (*http.Request, error) {
	valbs, err := cube.MarshalJSON(val)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(valbs))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	return req, nil
}

func DoAdminCall(cli *http.Client, checker IAdminChecker, req *http.Request) (*http.Response, error) {
	return checker.Do(req.Context(), cli, req)
}
