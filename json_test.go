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

func TestJsonStream(t *testing.T) {
	seq, err := JsonStreamZeroCopy("./test.json")
	if err != nil {
		t.Fatal(err)
	}
	for jv, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(">>>", jv, "<<<")
	}

	seq, err = JsonStreamZeroCopy("./test.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for jv, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(">>>", jv, "<<<")
	}
}
