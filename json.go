package cube

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"unsafe"
)

func MustMarshalJSON(v any) []byte {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	e := enc.Encode(v)
	if e != nil {
		panic(e)
	}
	return bytes.TrimSpace(buf.Bytes())
}

func MustMarshalJSONString(v any) string {
	bs, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(bs)
}

func MustMarshalJSONIndent(v any) []byte {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return bs
}

func MustMarshalJSONIndentString(v any) string {
	return string(MustMarshalJSONIndent(v))
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

func UnmarshalJSONString(txt string, dest any) error {
	err := json.Unmarshal(unsafe.Slice(unsafe.StringData(txt), len(txt)), dest)
	if err != nil {
		return err
	}
	return nil
}

func UnmarshalJSON(bytes []byte, dest any) error {
	err := json.Unmarshal(bytes, dest)
	if err != nil {
		return err
	}
	return nil
}

func MarshalJSON(val any) ([]byte, error) {
	return json.Marshal(val)
}

func MarshalJSONString(val any) (string, error) {
	bs, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}
