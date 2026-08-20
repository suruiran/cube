package action

import (
	"database/sql"
	"encoding/json"
	"net/http"

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

type PlainOutput struct {
	code   int
	header http.Header
	Txt    []byte
}

func NewPlainTextOutput(txt []byte) *PlainOutput {
	return &PlainOutput{Txt: txt}
}

func (p *PlainOutput) WithCode(code int) *PlainOutput {
	p.code = code
	return p
}

func (p *PlainOutput) WithHeader(fnc func(http.Header)) *PlainOutput {
	if p.header == nil {
		p.header = make(http.Header)
	}
	fnc(p.header)
	return p
}

func (p *PlainOutput) BytesBody() ([]byte, bool) {
	return p.Txt, true
}

func (p *PlainOutput) Code() int {
	return p.code
}

func (p *PlainOutput) Headers() http.Header {
	return p.header
}

var _ IHttpOutput = (*PlainOutput)(nil)

type JsonBytesOutput struct {
	Txt []byte
}

func NewJsonBytesOutput[T any](val T) (*JsonBytesOutput, error) {
	txt, err := cube.MarshalJSON(val)
	if err != nil {
		return nil, err
	}
	return &JsonBytesOutput{
		Txt: txt,
	}, nil
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
