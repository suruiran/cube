package ccli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/suruiran/cube"
)

type Items[T any] struct {
	Items  []T
	Decode func(string) (T, error)
	Encode func(*T) (string, error)
}

func (i *Items[T]) Set(val string) error {
	if i.Decode == nil {
		i.Decode = func(s string) (T, error) {
			var v T
			return v, cube.UnmarshalText(s, &v)
		}
	}

	item, err := i.Decode(val)
	if err != nil {
		return err
	}
	i.Items = append(i.Items, item)
	return nil
}

func (i *Items[T]) String() string {
	if i.Encode == nil {
		i.Encode = func(t *T) (string, error) { return cube.MarshalText(t) }
	}

	items := make([]string, 0, len(i.Items))
	for _, item := range i.Items {
		is, err := i.Encode(&item)
		if err != nil {
			is = fmt.Sprintf("MarshalFailed{%v, %s}", item, err.Error())
		}
		items = append(items, is)
	}
	return strings.Join(items, ", ")
}

var _ flag.Value = (*Items[int])(nil)
