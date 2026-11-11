package cube

import (
	"bufio"
	"compress/gzip"
	"encoding/json/jsontext"
	"errors"
	"io"
	"iter"
	"os"
	"strings"
)

var (
	ErrJsonStreamBadElement = errors.New("cube.jsonstream: bad element, expected `{}`")
)

func JsonStreamZeroCopy(fp string) (iter.Seq2[jsontext.Value, error], error) {
	if _, err := os.Stat(fp); err != nil {
		return nil, err
	}
	return func(yield func(jsontext.Value, error) bool) {
		fobj, err := os.Open(fp)
		if err != nil {
			yield(nil, err)
			return
		}
		defer fobj.Close() //nolint: errcheck

		var reader io.Reader = bufio.NewReaderSize(fobj, 4*1024*1024)

		if strings.HasSuffix(strings.ToLower(fp), ".gz") {
			gr, err := gzip.NewReader(reader)
			if err != nil {
				yield(nil, err)
				return
			}
			defer gr.Close() //nolint: errcheck
			reader = gr
		}

		dec := jsontext.NewDecoder(reader)

		switch dec.PeekKind() {
		case jsontext.KindBeginArray:
			{
				_, err := dec.ReadToken()
				if err != nil {
					yield(nil, err)
					return
				}
				for dec.PeekKind() != jsontext.KindEndArray {
					jv, err := dec.ReadValue()
					if err != nil {
						yield(nil, err)
						return
					}
					if len(jv) < 2 || jv[0] != '{' || jv[len(jv)-1] != '}' {
						yield(nil, ErrJsonStreamBadElement)
						return
					}
					if !yield(jv, nil) {
						return
					}
				}
				if _, err := dec.ReadToken(); err != nil {
					yield(nil, err)
					return
				}
			}
		case jsontext.KindBeginObject:
			{
				for {
					jv, err := dec.ReadValue()
					if err != nil {
						if errors.Is(err, io.EOF) {
							return
						}
						yield(nil, err)
						return
					}
					if len(jv) < 2 || jv[0] != '{' || jv[len(jv)-1] != '}' {
						yield(nil, ErrJsonStreamBadElement)
						return
					}
					if !yield(jv, nil) {
						return
					}
				}
			}
		default:
			{
				yield(nil, ErrJsonStreamBadElement)
				return
			}
		}
	}, nil
}
