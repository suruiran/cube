package sqlx

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type IndexField struct {
	Name string
	Desc bool

	defineIdx int
	order     int
}

type Index struct {
	Name   string
	Unique bool
	Fields []*IndexField
}

// Indexes returns the indexes of the model.
// index tag format:
//
//	`[!↓>]<indexname>[/<orderidx>]`
//
// - `!` means the index is unique.
// - `↓` or `>` means the field is sorted in descending order.
// - `indexname` is the name of the index.
// - `orderidx` is the order index of the field in the index.
func (info *ModelInfo) Indexes(keys ...string) ([]*Index, error) {
	indexes := map[string]*Index{}
	for _, fv := range info.Fields {
		for _, item := range fv.Opts.GetAll(_keys(keys, defaultindexkeys)...) {
			if err := parseIndex(fv.Name, indexes, item); err != nil {
				return nil, err
			}
		}
	}

	indexesv := make([]*Index, 0, len(indexes))
	for _, index := range indexes {
		sort.Slice(
			index.Fields,
			func(i int, j int) bool {
				a := index.Fields[i]
				b := index.Fields[j]
				if a.order != b.order {
					return a.order < b.order
				}
				return a.defineIdx < b.defineIdx
			},
		)
		indexesv = append(indexesv, index)
	}
	sort.Slice(
		indexesv,
		func(i int, j int) bool {
			a := indexesv[i]
			b := indexesv[j]
			return a.Name < b.Name
		},
	)
	return indexesv, nil
}

func parseIndex(name string, indexes map[string]*Index, txt string) error {
	txt = strings.TrimSpace(txt)
	field := &IndexField{
		Name: name,
	}

	unique := false
loop:
	for {
		runes := []rune(txt)
		if len(runes) < 1 {
			return fmt.Errorf("sqlx: index field `%s` must have a index name", txt)
		}
		switch runes[0] {
		case '!':
			{
				unique = true
				txt = strings.TrimSpace(string(runes[1:]))
			}
		case '>', '↓':
			{
				field.Desc = true
				txt = strings.TrimSpace(string(runes[1:]))
			}
		default:
			{
				break loop
			}
		}
	}

	if txt == "" {
		return fmt.Errorf("sqlx: index field `%s` must have a index name", txt)
	}

	order := 0
	indexname := ""
	orderidx := strings.LastIndex(txt, "/")
	if orderidx < 0 {
		indexname = txt
	} else {
		indexname = strings.TrimSpace(txt[:orderidx])
		_iv, err := strconv.ParseInt(strings.TrimSpace(txt[orderidx+1:]), 10, 64)
		if err != nil {
			return fmt.Errorf("sqlx: index field `%s` must have a define index", txt)
		}
		order = int(_iv)
	}
	field.order = order

	index := indexes[indexname]
	if index == nil {
		if indexname == "" {
			return fmt.Errorf("sqlx: index field `%s` must have a index name", txt)
		}
		index = &Index{
			Name: indexname,
		}
		indexes[indexname] = index
	} else {
		if index.Unique != unique {
			return fmt.Errorf("sqlx: index `%s` has different unique definition", index.Name)
		}
	}
	index.Unique = unique
	field.defineIdx = len(index.Fields)
	index.Fields = append(index.Fields, field)
	return nil
}
