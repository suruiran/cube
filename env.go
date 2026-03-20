package cube

import (
	"fmt"
	"os"
)

func env[T any](key string) (T, error) {
	var t T
	value, ok := os.LookupEnv(key)
	if !ok {
		return t, fmt.Errorf("cube.env: `%s` not found", key)
	}
	if value == "" {
		return t, fmt.Errorf("cube.env: `%s` is empty", key)
	}
	if err := UnmarshalText(value, &t); err != nil {
		return t, fmt.Errorf("cube.env: `%s` unmarshal failed: %w", key, err)
	}
	return t, nil
}

func MustEnv[T any](key string) T {
	val, err := env[T](key)
	if err != nil {
		panic(err)
	}
	return val
}

func Env[T any](key string, fallback T) T {
	val, err := env[T](key)
	if err != nil {
		return fallback
	}
	return val
}
