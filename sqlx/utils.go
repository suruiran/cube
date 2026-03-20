package sqlx

import (
	"reflect"
	"strings"
)

func UnwrapSqlNullType(st reflect.Type) (reflect.Type, bool) {
	if st.PkgPath() == "database/sql" && strings.HasPrefix(st.Name(), "Null") && st.Kind() == reflect.Struct && st.NumField() == 2 {
		fv := st.Field(0)
		fvalid := st.Field(1)
		ok := fvalid.Name == "Valid" && fvalid.Type.Kind() == reflect.Bool
		return fv.Type, ok
	}
	return nil, false
}
