package cube

import (
	"context"
	"errors"
	"io"
	"iter"
	"runtime"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidUTF8 = errors.New("invalid UTF-8 encoding")
)

func rune_seq_internal(ctx context.Context, r io.Reader, nocopy bool, bufsize int) iter.Seq2[[]rune, error] {
	return func(yield func([]rune, error) bool) {
		buf := make([]byte, bufsize)
		runes := make([]rune, 0, bufsize)
		swap := make([]byte, 0, utf8.UTFMax)
		loopc := 0

		for {
			loopc++
			if loopc&0xF == 0 {
				select {
				case <-ctx.Done():
					yield(nil, ctx.Err())
					return
				default:
				}
			}

			copy(buf, swap)
			swaplen := len(swap)
			n, err := r.Read(buf[swaplen:])
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
			if n > 0 {
				swap = swap[:0]
				bufsize := n + swaplen
				runes = runes[:0]
				for i := 0; i < bufsize; {
					char, size := utf8.DecodeRune(buf[i:bufsize])
					if char == utf8.RuneError && size == 1 {
						if !utf8.FullRune(buf[i:bufsize]) {
							swap = append(swap, buf[i:bufsize]...)
							break
						}
						yield(nil, ErrInvalidUTF8)
						return
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
			} else {
				runtime.Gosched()
			}
		}
	}
}

func RuneSeq(ctx context.Context, r io.Reader, bufsize int) iter.Seq2[[]rune, error] {
	return rune_seq_internal(ctx, r, false, bufsize)
}

func RuneSeqNoCopy(ctx context.Context, r io.Reader, bufsize int) iter.Seq2[[]rune, error] {
	return rune_seq_internal(ctx, r, true, bufsize)
}

func SkipBOM(input iter.Seq2[[]rune, error]) iter.Seq2[[]rune, error] {
	return func(yield func([]rune, error) bool) {
		fc := false
		for chars, err := range input {
			if !fc {
				if err != nil {
					yield(nil, err)
					return
				}
				if len(chars) < 1 {
					continue
				}
				if chars[0] == 0xFEFF {
					fc = true
					if len(chars) > 1 {
						if !(yield(chars[1:], nil)) {
							return
						}
					}
				} else {
					fc = true
					if !(yield(chars, nil)) {
						return
					}
				}
				continue
			}
			if !yield(chars, err) {
				return
			}
		}
	}
}

func HeadRunes(txt string, size int) (string, error) {
	buf := make([]rune, 0, size)
	for rs, err := range RuneSeqNoCopy(context.Background(), strings.NewReader(txt), size) {
		if err != nil {
			return "", err
		}
		remain := size - len(buf)
		if len(rs) > remain {
			rs = rs[:remain]
		}
		buf = append(buf, rs...)
		if len(buf) >= size {
			break
		}
	}
	return string(buf), nil
}
