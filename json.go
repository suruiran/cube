package cube

import (
	"encoding/json"
	"reflect"
	"strconv"
	"unsafe"
)

func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func UnmarshalJSON(input []byte, dest any) error {
	return json.Unmarshal(input, dest)
}

func MarshalJSONIndent(v any) ([]byte, error) {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func MarshalJSONString(val any) (string, error) {
	bs, err := MarshalJSON(val)
	if err != nil {
		return "", err
	}
	return unsafe.String(unsafe.SliceData(bs), len(bs)), nil
}

func MarshalJSONIndentString(v any) (string, error) {
	bs, err := MarshalJSONIndent(v)
	if err != nil {
		return "", err
	}
	return unsafe.String(unsafe.SliceData(bs), len(bs)), nil
}

func UnmarshalJSONString(txt string, dest any) error {
	err := UnmarshalJSON(unsafe.Slice(unsafe.StringData(txt), len(txt)), dest)
	if err != nil {
		return err
	}
	return nil
}

type IJsonValue interface {
	~int64 | ~bool | ~string | ~float64
}

func Peek[T IJsonValue](mapv map[string]any, keys ...string) (T, bool) {
	var dv T

	var cur any = mapv
	for _, key := range keys {
		switch node := cur.(type) {
		case map[string]any:
			{
				var ok bool
				cur, ok = node[key]
				if !ok {
					return dv, false
				}
			}
		case []any:
			{
				iv, err := strconv.ParseUint(key, 10, 64)
				if err != nil {
					return dv, false
				}
				if iv > uint64(len(node)-1) {
					return dv, false
				}
				cur = node[iv]
			}
		default:
			{
				return dv, false
			}
		}
	}

	dvv := reflect.ValueOf(&dv).Elem()
	switch dvv.Kind() {
	case reflect.Int64:
		{
			fv, ok := cur.(float64)
			if ok {
				dvv.SetInt(int64(fv))
				return dv, true
			}
		}
	case reflect.Float64:
		{
			fv, ok := cur.(float64)
			if ok {
				dvv.SetFloat(fv)
				return dv, true
			}
		}
	case reflect.String:
		{
			sv, ok := cur.(string)
			if ok {
				dvv.SetString(sv)
				return dv, true
			}
		}
	case reflect.Bool:
		{
			bv, ok := cur.(bool)
			if ok {
				dvv.SetBool(bv)
				return dv, true
			}
		}
	}
	return dv, false
}
