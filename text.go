package cube

import (
	"reflect"
	"strconv"
)

type ITextUnmarshaler interface {
	UnmarshalText(text string) error
}

type ITextMarshaler interface {
	MarshalText(val any) (string, error)
}

func UnmarshalText(text string, ptr any) error {
	if u, ok := ptr.(ITextUnmarshaler); ok {
		return u.UnmarshalText(text)
	}

	if _sptr, ok := ptr.(*string); ok {
		*_sptr = text
		return nil
	}

	vv := reflect.ValueOf(ptr).Elem()
	switch vv.Kind() {
	case reflect.String:
		{
			vv.SetString(text)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		{
			iv, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return err
			}
			vv.SetInt(iv)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		{
			uv, err := strconv.ParseUint(text, 10, 64)
			if err != nil {
				return err
			}
			vv.SetUint(uv)
		}
	case reflect.Float32, reflect.Float64:
		{
			fv, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return err
			}
			vv.SetFloat(fv)
		}
	case reflect.Bool:
		{
			bv, err := strconv.ParseBool(text)
			if err != nil {
				return err
			}
			vv.SetBool(bv)
		}
	default:
		{
			return UnmarshalJSONString(text, ptr)
		}
	}
	return nil
}

func MarshalText(val any) (string, error) {
	if m, ok := val.(ITextMarshaler); ok {
		return m.MarshalText(val)
	}

	switch v := val.(type) {
	case string:
		{
			return v, nil
		}
	case *string:
		{
			if v == nil {
				return "", nil
			}
			return *v, nil
		}
	}

	vv := reflect.ValueOf(val)
	if vv.Kind() == reflect.Pointer {
		vv = vv.Elem()
	}

	switch vv.Kind() {
	case reflect.String:
		{
			return vv.String(), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		{
			return strconv.FormatInt(vv.Int(), 10), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		{
			return strconv.FormatUint(vv.Uint(), 10), nil
		}
	case reflect.Float32, reflect.Float64:
		{
			return strconv.FormatFloat(vv.Float(), 'f', -1, 64), nil
		}
	case reflect.Bool:
		{
			return strconv.FormatBool(vv.Bool()), nil
		}
	default:
		{
			return MarshalJSONString(val)
		}
	}
}
