package cube

import (
	"fmt"
	"testing"
)

func TestJSON(t *testing.T) {
	fmt.Println(Must(MarshalJSONIndentString(map[any]any{
		12:    "ddd",
		"xxx": "xxx",
	})))
}
