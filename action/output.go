package action

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"unsafe"

	"github.com/suruiran/cube"
)

type IHttpOutput interface {
	Code() int
	Headers() http.Header
	BytesBody() ([]byte, bool)
}

type Output[T any] struct {
	code   int
	header http.Header
	val    sql.Null[T]
}

func (c *Output[T]) BytesBody() ([]byte, bool) {
	return nil, false
}

var (
	jsonNull = []byte("null")
)

func (c *Output[T]) MarshalJSON() ([]byte, error) {
	if c.val.Valid {
		return cube.MarshalJSON(c.val.V)
	}
	return jsonNull, nil
}

func (c *Output[T]) Code() int {
	return c.code
}

func (c *Output[T]) Headers() http.Header {
	return c.header
}

var _ IHttpOutput = (*Output[int])(nil)

var _ json.Marshaler = (*Output[int])(nil)

func NewOutput[T any](val T) *Output[T] {
	return &Output[T]{
		code: http.StatusOK,
		val:  sql.Null[T]{V: val, Valid: true},
	}
}

func (c *Output[T]) WithCode(code int) *Output[T] {
	c.code = code
	return c
}

func (c *Output[T]) WithHeader(fnc func(http.Header)) *Output[T] {
	if c.header == nil {
		c.header = make(http.Header)
	}
	fnc(c.header)
	return c
}

type PlainTextOutput struct {
	Txt string
}

func NewPlainTextOutput(txt string) *PlainTextOutput {
	return &PlainTextOutput{Txt: txt}
}

func (p *PlainTextOutput) BytesBody() ([]byte, bool) {
	return unsafe.Slice(unsafe.StringData(p.Txt), len(p.Txt)), true
}

func (p *PlainTextOutput) Code() int {
	return 200
}

func (p *PlainTextOutput) Headers() http.Header {
	return nil
}

var _ IHttpOutput = (*PlainTextOutput)(nil)

type JsonBytesOutput struct {
	Txt []byte
}

func NewJsonBytesOutput[T any](val T) *JsonBytesOutput {
	return &JsonBytesOutput{
		Txt: cube.MustMarshalJSON(val),
	}
}

func (j *JsonBytesOutput) BytesBody() ([]byte, bool) {
	return j.Txt, true
}

func (j *JsonBytesOutput) Code() int {
	return 200
}

var (
	_JsonHeader = make(http.Header)
)

func init() {
	_JsonHeader.Set("Content-Type", "application/json")
}

func (j *JsonBytesOutput) Headers() http.Header {
	return _JsonHeader
}

var _ IHttpOutput = (*JsonBytesOutput)(nil)
