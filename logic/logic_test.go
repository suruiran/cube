package logic

import (
	"fmt"
	"testing"
)

func TestLazyBool(t *testing.T) {
	type C struct {
		C1 int
	}

	type B struct {
		B1 int
		C  *C
	}
	type A struct {
		B *B
	}

	var a A
	fmt.Println(All(Lazy(func() int { return a.B.C.C1 })))
}
