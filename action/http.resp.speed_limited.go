package action

import (
	"context"
	"math/rand/v2"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type SpeedLimitedRespWriter struct {
	ctx    context.Context
	Writer http.ResponseWriter
	*rate.Limiter
}

func NewSpeedLimitedRespWriter(ctx context.Context, writer http.ResponseWriter, kbs int) http.ResponseWriter {
	if kbs <= 0 {
		panic("kbs must be greater than 0")
	}
	return &SpeedLimitedRespWriter{
		ctx:     ctx,
		Writer:  writer,
		Limiter: rate.NewLimiter(rate.Limit(kbs*1024), kbs*1024),
	}
}

func (s *SpeedLimitedRespWriter) Header() http.Header {
	return s.Writer.Header()
}

func (s *SpeedLimitedRespWriter) Write(p []byte) (int, error) {
	w := 0
	r := len(p)
	b := s.Burst()
	for {
		time.Sleep(time.Duration(rand.IntN(500)) * time.Millisecond)
		if r <= b {
			if err := s.WaitN(s.ctx, r); err != nil {
				return w, err
			}
			n, err := s.Writer.Write(p[w:])
			w += n
			return w, err
		}

		if err := s.WaitN(s.ctx, b); err != nil {
			return w, err
		}
		n, err := s.Writer.Write(p[w : w+b])
		w += n
		r -= n
		if err != nil {
			return w, err
		}
	}
}

func (s *SpeedLimitedRespWriter) WriteHeader(statusCode int) {
	s.Writer.WriteHeader(statusCode)
}

var _ http.ResponseWriter = (*SpeedLimitedRespWriter)(nil)
