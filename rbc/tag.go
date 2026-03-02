package rbc

import (
	"fmt"
	"net/url"
	"strings"
)

type Tag struct {
	Name string
	Opts TagOptions
}

func parsetag(txt string) *Tag {
	tag := &Tag{
		Opts: TagOptions{
			Values: make(url.Values),
		},
	}

	name := ""
	nameidx := strings.IndexByte(txt, ',')
	if nameidx > -1 {
		name = strings.TrimSpace(txt[:nameidx])
		txt = strings.TrimSpace(txt[nameidx+1:])
	} else {
		name = strings.TrimSpace(txt)
		txt = ""
	}
	tag.Name = name

	if txt != "" {
		for opt := range strings.SplitSeq(txt, ";") {
			opt = strings.TrimSpace(opt)
			if before, after, ok := strings.Cut(opt, "="); ok {
				tag.Opts.Add(strings.TrimSpace(before), strings.TrimSpace(after))
			} else {
				tag.Opts.Add(opt, "")
			}
		}
	}
	return tag
}

var (
	tagDefaultNameConverters = map[string]func(string) string{}
)

func RegisterTagNameConverter(tagname string, converter func(string) string) {
	tagDefaultNameConverters[tagname] = converter
}

func (ti *TypeInfo) inittag(tagname string) error {
	if ti.tagsmap == nil {
		ti.tagsmap = make(map[string]map[string]*FieldWithTag)
	}
	if ti.tagslst == nil {
		ti.tagslst = make(map[string][]*FieldWithTag)
	}

	if _, ok := ti.tagsmap[tagname]; ok {
		return nil
	}

	lst := []*FieldWithTag{}
	mv := make(map[string]*FieldWithTag)

	for _, fv := range ti.fields {
		tagtxt := strings.TrimSpace(fv.Info.Tag.Get(tagname))
		if tagtxt == "-" {
			continue
		}

		tag := parsetag(tagtxt)
		if tag.Name == "" {
			if converter, ok := tagDefaultNameConverters[tagname]; ok {
				tag.Name = converter(tag.Name)
			} else {
				tag.Name = fv.Info.Name
			}
		}

		if _, exist := mv[tag.Name]; exist {
			return fmt.Errorf("sqlx: tag `%s` duplicated in type `%s`", tag.Name, ti.Type.Name())
		}

		lst = append(lst, &FieldWithTag{
			Field: fv,
			Tag:   tag,
		})
		mv[tag.Name] = lst[len(lst)-1]
	}

	ti.tagsmap[tagname] = mv
	ti.tagslst[tagname] = lst

	return nil
}
