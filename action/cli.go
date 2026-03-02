package action

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/suruiran/cube"
)

func NewJsonRequest[T any](ctx context.Context, method string, url string, val T) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(cube.MustJsonMarshal(val)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	return req, nil
}

func DoAdminCall(req *http.Request, authdir string) (*http.Response, error) {
	filename := uuid.NewString()
	code := cube.RandAsciiBytes(16)
	fp := filepath.Join(authdir, filename)
	werr := os.WriteFile(fp, code, 0600)
	if werr != nil {
		return nil, werr
	}
	req.Header.Add("X-Admin-Auth", fmt.Sprintf("%s/%s", filename, code))
	return http.DefaultClient.Do(req)
}
