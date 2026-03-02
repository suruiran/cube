package action

import (
	"database/sql"
	"net/http"

	"github.com/goccy/go-json"
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

// MarshalJSON implements json.Marshaler.
func (c *Output[T]) MarshalJSON() ([]byte, error) {
	if c.val.Valid {
		return json.Marshal(c.val.V)
	}
	return json.Marshal(nil)
}

// Code implements IHttpOutput.
func (c *Output[T]) Code() int {
	return c.code
}

// Headers implements IHttpOutput.
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
	return []byte(p.Txt), true
}

func (p *PlainTextOutput) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (p *PlainTextOutput) Code() int {
	return 200
}

func (p *PlainTextOutput) Headers() http.Header {
	return nil
}

var _ IHttpOutput = (*PlainTextOutput)(nil)
var _ json.Marshaler = (*PlainTextOutput)(nil)

type JsonBytesOutput struct {
	Txt []byte
}

func NewJsonBytesOutput[T any](val T) *JsonBytesOutput {
	return &JsonBytesOutput{
		Txt: cube.MustJsonMarshal(val),
	}
}

func (j *JsonBytesOutput) MarshalJSON() ([]byte, error) {
	return nil, nil
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
var _ json.Marshaler = (*JsonBytesOutput)(nil)
