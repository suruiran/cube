package action

import (
	"bytes"
	"context"
	"net/http"

	"github.com/suruiran/cube"
)

func NewJsonRequest[T any](ctx context.Context, method string, url string, val T) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(cube.MustMarshalJSON(val)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	return req, nil
}

func DoAdminCall(cli *http.Client, checker IAdminChecker, req *http.Request) (*http.Response, error) {
	return checker.Do(req.Context(), cli, req)
}
