package cube

import (
	"context"
	"errors"
	"io"
	"iter"
	"unicode/utf8"
)

var (
	RuneReadBufferSize = 1024
	ErrInvalidUTF8     = errors.New("invalid UTF-8 encoding")
)

func rune_seq_internal(ctx context.Context, r io.Reader, nocopy bool) iter.Seq2[[]rune, error] {
	return func(yield func([]rune, error) bool) {
		buf := make([]byte, RuneReadBufferSize)
		runes := make([]rune, 0, RuneReadBufferSize)
		swap := make([]byte, 0, utf8.UTFMax)

		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			copy(buf, swap)
			swaplen := len(swap)
			n, err := r.Read(buf[swaplen:])
			if n > 0 {
				swap = swap[:0]
				bufsize := n + swaplen
				runes = runes[:0]
				for i := 0; i < bufsize; {
					remainsize := bufsize - i
					char, size := utf8.DecodeRune(buf[i:bufsize])
					if char == utf8.RuneError && size == 1 {
						if remainsize < utf8.UTFMax {
							swap = append(swap, buf[i:bufsize]...)
							break
						}
						yield(nil, ErrInvalidUTF8)
						return
					}
					if remainsize < size {
						swap = append(swap, buf[i:bufsize]...)
						break
					}
					runes = append(runes, char)
					i += size
				}

				if len(runes) > 0 {
					if nocopy {
						if !yield(runes, nil) {
							return
						}
					} else {
						var tmp = make([]rune, len(runes))
						copy(tmp, runes)
						if !yield(tmp, nil) {
							return
						}
					}
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					if len(swap) > 0 {
						yield(nil, io.ErrUnexpectedEOF)
						return
					}
				}
				yield(nil, err)
				return
			}
		}
	}
}

func RuneSeq(ctx context.Context, r io.Reader) iter.Seq2[[]rune, error] {
	return rune_seq_internal(ctx, r, false)
}

func RuneSeqNoCopy(ctx context.Context, r io.Reader) iter.Seq2[[]rune, error] {
	return rune_seq_internal(ctx, r, true)
}
