package cube

import (
	"fmt"
	"regexp"
)

type SingedInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type UnsignedInt interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type IntType interface {
	SingedInt | UnsignedInt
}

type IEnum interface {
	IntType
	fmt.Stringer
}

var (
	enumStringRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
)

func AllIntEnums[T IEnum](min T, max T, regexps ...*regexp.Regexp) []T {
	var reg *regexp.Regexp
	if len(regexps) > 0 {
		reg = regexps[0]
	} else {
		reg = enumStringRegex
	}

	items := make([]T, 0, 16)
	for i := min; i <= max; {
		if reg.MatchString(i.String()) {
			items = append(items, i)
		}
		i++
	}
	return items
}
