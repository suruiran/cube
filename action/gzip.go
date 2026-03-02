package action

import (
	"bytes"
	"compress/gzip"
	"net/http"
)

type GzippedOutput struct {
	code    int
	content []byte
	header  http.Header
}

func (obj *GzippedOutput) BytesBody() ([]byte, bool) {
	return obj.content, true
}

func (obj *GzippedOutput) Code() int {
	return obj.code
}

func (obj *GzippedOutput) Headers() http.Header {
	return obj.header
}

func NewGzippedOutput(code int, content []byte, contentType string) *GzippedOutput {
	obj := &GzippedOutput{
		code:    code,
		content: content,
		header:  make(http.Header),
	}
	obj.header.Set("content-type", contentType)
	if len(content) > 2048 {
		buf := bytes.NewBuffer(make([]byte, 0, 4096))
		w := gzip.NewWriter(buf)
		_, _ = w.Write(obj.content)
		_ = w.Flush()
		_ = w.Close()
		obj.content = buf.Bytes()
		obj.header.Set("content-encoding", "gzip")
	}
	return obj
}

var _ IHttpOutput = (*GzippedOutput)(nil)
