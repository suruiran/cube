package cube

import (
	"fmt"
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	_ = os.Setenv("TEST", "123")

	fmt.Println(Env("TEST", 0))

	_ = os.Setenv("TEST", "true")

	fmt.Println(Env("TEST", false))

	_ = os.Setenv("TEST", "true")

	fmt.Println(Env("TEST", ""))
}
