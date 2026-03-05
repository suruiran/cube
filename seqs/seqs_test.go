package seqs

import (
	"context"
	"strconv"
	"testing"
)

func TestPipe(t *testing.T) {
	var input = []int{1, 2, 3, 4, 5}

	for ele := range Pipe[int, string](
		t.Context(),
		Slice(input),
		Op(func(ctx context.Context, v int) (string, Kind) { return strconv.Itoa(v), Ok }),
	) {
		t.Log(ele)
	}
}
