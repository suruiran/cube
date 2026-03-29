package cube

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestRuneSeq(t *testing.T) {
	val := strings.Repeat("A", 1023) + "我"
	fmt.Println(len(val), len([]byte(val)), len([]rune(val)))

	r := strings.NewReader(val)
	seq := RuneSeq(t.Context(), r, 512)

	for runes, err := range seq {
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			t.Fatal(err)
		}
		t.Log(string(runes))
	}
}
