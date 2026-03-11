package cube

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"iter"
	"os"
	"strings"

	"encoding/json"
)

var (
	ErrJsonStreamBadElement = errors.New("cube.json.stream: bad element, required '{'")
)

type _CloseScope struct {
	file func() error
	gzip func() error
}

func (cs *_CloseScope) Close() {
	if cs.file != nil {
		cs.file() //nolint:errcheck
	}
	if cs.gzip != nil {
		cs.gzip() //nolint:errcheck
	}
}

func JsonStreamZeroCopy(fp string) (iter.Seq2[json.RawMessage, error], error) {
	fobj, fopen_err := os.Open(fp)
	if fopen_err != nil {
		return nil, fopen_err
	}

	var cs = _CloseScope{file: fobj.Close}

	var reader io.Reader = bufio.NewReaderSize(fobj, 1024*1024*4)

	fp = strings.ToLower(fp)
	isgzip := strings.HasSuffix(fp, ".gz")
	isjsonl := false
	if isgzip {
		isjsonl = strings.HasSuffix(fp, ".jsonl.gz")
	} else {
		isjsonl = strings.HasSuffix(fp, ".jsonl")
	}
	if isjsonl {
		cs.file = nil
		cs.Close()
		return jsonlStreamZeroCopy(fobj, isgzip, false)
	}

	if isgzip {
		gr, gr_init_err := gzip.NewReader(reader)
		if gr_init_err != nil {
			return nil, gr_init_err
		}
		reader = gr
		cs.gzip = gr.Close
	}

	dec := json.NewDecoder(reader)
	token, token_read_err := dec.Token()
	if token_read_err != nil {
		return nil, token_read_err
	}
	if token != json.Delim('[') {
		cs.file = nil
		cs.Close()
		return jsonlStreamZeroCopy(fobj, isgzip, true)
	}

	var v json.RawMessage
	return func(yield func(json.RawMessage, error) bool) {
		defer cs.Close()

		for dec.More() {
			v = v[:0]
			err := dec.Decode(&v)
			if err != nil {
				yield(nil, err)
				return
			}
			if len(v) < 1 || v[0] != '{' || v[len(v)-1] != '}' {
				yield(nil, ErrJsonStreamBadElement)
				return
			}
			if !yield(v, nil) {
				return
			}
		}
	}, nil
}

const (
	jsonl_max_line_size     = 128 * 1024 * 1024
	jsonl_default_line_size = 1024 * 1024
)

func jsonlStreamZeroCopy(fobj *os.File, isgzip bool, seek0 bool) (iter.Seq2[json.RawMessage, error], error) {
	cs := _CloseScope{
		file: fobj.Close,
	}

	if seek0 {
		_, seek_err := fobj.Seek(0, io.SeekStart)
		if seek_err != nil {
			cs.Close()
			return nil, seek_err
		}
	}

	var reader io.Reader = bufio.NewReaderSize(fobj, 1024*1024*4)
	if isgzip {
		gr, gziperr := gzip.NewReader(reader)
		if gziperr != nil {
			cs.Close()
			return nil, gziperr
		}
		reader = gr
		cs.gzip = gr.Close
	}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, jsonl_default_line_size)
	scanner.Buffer(buf, jsonl_max_line_size)

	return func(yield func(json.RawMessage, error) bool) {
		defer cs.Close()

		for scanner.Scan() {
			v := bytes.TrimSpace(scanner.Bytes())
			if len(v) == 0 {
				continue
			}
			if v[0] != '{' || v[len(v)-1] != '}' {
				yield(nil, ErrJsonStreamBadElement)
				return
			}
			if !yield(v, nil) {
				return
			}
		}
	}, nil
}
