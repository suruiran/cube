package logx

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
)

type Opts struct {
	Filename         string `json:"filename" toml:"filename"`
	MultiProcessSafe bool   `json:"multi_process_safe" toml:"multi_process_safe"`

	Level       slog.Level `json:"level" toml:"level"`
	AddSource   bool       `json:"add_source" toml:"add_source"`
	WithStdout  bool       `json:"with_stdout" toml:"with_stdout"`
	StdoutLevel slog.Level `json:"stdout_level" toml:"stdout_level"`

	BufferSize int `json:"buffer_size" toml:"buffer_size"`

	Rolling *RollingOptions `json:"rolling" toml:"rolling"`

	Forwards []slog.Handler `json:"-" toml:"-"`
}

func New(opts *Opts) (*slog.Logger, error) {
	nobuff := opts.BufferSize < 0
	if opts.Rolling != nil && nobuff {
		return nil, errors.New("cube.logx: no_buffered is not supported with rolling")
	}
	handlers := []slog.Handler{}
	var lw io.Writer
	if opts.Rolling != nil {
		if opts.Rolling.BufferSize <= 0 {
			opts.Rolling.BufferSize = opts.BufferSize
		}
		if opts.MultiProcessSafe {
			opts.Rolling.OpenFile = func(s string, i int, fm os.FileMode) (IFile, error) {
				return OpenLockFile(s, i, fm)
			}
		} else {
			opts.Rolling.OpenFile = func(s string, i int, fm os.FileMode) (IFile, error) {
				return os.OpenFile(s, i, fm)
			}
		}
		rf, err := NewRollingFile(opts.Filename, opts.Rolling)
		if err != nil {
			return nil, err
		}
		lw = NewAutoSaveWriter(rf)
	} else {
		fcf := os.O_CREATE | os.O_WRONLY | os.O_APPEND
		if nobuff {
			fcf |= os.O_SYNC
		}

		var fobj io.WriteCloser
		if opts.MultiProcessSafe {
			var err error
			fobj, err = OpenLockFile(opts.Filename, fcf, 0644)
			if err != nil {
				return nil, err
			}
		} else {
			var err error
			fobj, err = os.OpenFile(opts.Filename, fcf, 0644)
			if err != nil {
				return nil, err
			}
		}
		if nobuff {
			lw = NewAutoSaveWriter(NewNoBufferedWriter(fobj), fobj.Close)
		} else {
			lw = NewAutoSaveWriter(bufio.NewWriterSize(fobj, opts.BufferSize), fobj.Close)
		}
	}

	handlers = append(handlers, slog.NewJSONHandler(lw, &slog.HandlerOptions{Level: opts.Level, AddSource: opts.AddSource}))

	handlers = append(handlers, opts.Forwards...)
	if opts.WithStdout {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: opts.StdoutLevel, AddSource: opts.AddSource}))
	}

	var handler slog.Handler
	if len(handlers) > 1 {
		handler = slog.NewMultiHandler(handlers...)
	} else {
		handler = handlers[0]
	}
	return slog.New(handler), nil
}
