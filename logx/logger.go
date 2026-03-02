package logx

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
)

type Opts struct {
	Filename string `json:"filename" toml:"filename"`
	LockFile bool   `json:"lock_file" toml:"lock_file"`

	Level       slog.Level `json:"level" toml:"level"`
	AddSource   bool       `json:"add_source" toml:"add_source"`
	WithStdout  bool       `json:"with_stdout" toml:"with_stdout"`
	StdoutLevel slog.Level `json:"stdout_level" toml:"stdout_level"`

	BufferSize int `json:"buffer_size" toml:"buffer_size"`

	Rolling *RollingOptions `json:"rolling" toml:"rolling"`

	DisableLocal bool           `json:"disable_local" toml:"disable_local"`
	Forwards     []slog.Handler `json:"-" toml:"-"`
}

func New(opts *Opts) (*slog.Logger, error) {
	nobuff := opts.BufferSize < 0
	if opts.Rolling != nil && nobuff {
		return nil, errors.New("rrscpks.logger: no_buffered is not supported with rolling")
	}
	if opts.DisableLocal && len(opts.Forwards) == 0 {
		return nil, errors.New("rrscpks.logger: disable_local is true but no forwards")
	}

	handlers := []slog.Handler{}
	if !opts.DisableLocal {
		var lw io.Writer
		if opts.Rolling != nil {
			if opts.Rolling.BufferSize <= 0 {
				opts.Rolling.BufferSize = opts.BufferSize
			}
			if opts.LockFile {
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

			var fobj io.Writer
			if opts.LockFile {
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
				lw = NewNoBufferedWriter(fobj)
			} else {
				lw = NewAutoSaveWriter(bufio.NewWriterSize(fobj, opts.BufferSize))
			}
		}

		handlers = append(handlers, slog.NewJSONHandler(lw, &slog.HandlerOptions{Level: opts.Level, AddSource: opts.AddSource}))

	}

	handlers = append(handlers, opts.Forwards...)
	if opts.WithStdout {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: opts.StdoutLevel, AddSource: opts.AddSource}))
	}

	var handler slog.Handler
	if len(handlers) > 1 {
		handler = NewMultSender(handlers...)
	} else {
		handler = handlers[0]
	}
	return slog.New(handler), nil
}
