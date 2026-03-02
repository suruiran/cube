package logx

import (
	"fmt"
	"testing"
)

func TestLogger(t *testing.T) {
	txt := []byte(`""`)
	fmt.Println(string(txt[1:len(txt)-1]) == "")
}
