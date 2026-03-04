package ccli

import (
	"flag"
	"strings"
)

type Strings []string

func (s *Strings) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func (s *Strings) String() string {
	return strings.Join(*s, ",")
}

var (
	_ flag.Value = (*Strings)(nil)
)

type ItemList[T any] struct {
	Items  []T
	decode func(string) (T, error)
	encode func(*T) string
}

func NewFlagItems[T any](decode func(string) (T, error), encode func(*T) string) *ItemList[T] {
	return &ItemList[T]{
		decode: decode,
		encode: encode,
	}
}

func (i *ItemList[T]) Set(val string) error {
	item, err := i.decode(val)
	if err != nil {
		return err
	}
	i.Items = append(i.Items, item)
	return nil
}

func (i *ItemList[T]) String() string {
	items := make([]string, 0, len(i.Items))
	for _, item := range i.Items {
		items = append(items, i.encode(&item))
	}
	return strings.Join(items, ",")
}

var _ flag.Value = (*ItemList[int])(nil)
