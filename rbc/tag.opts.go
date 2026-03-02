package rbc

import (
	"net/url"
	"slices"
	"strconv"
)

type TagOptions struct {
	url.Values
}

func (opts *TagOptions) GetInt(key string) (int, error) {
	iv, err := strconv.ParseInt(opts.Get(key), 10, 64)
	return int(iv), err
}

func (opts *TagOptions) GetUint(key string) (uint, error) {
	iv, err := strconv.ParseUint(opts.Get(key), 10, 64)
	return uint(iv), err
}

func (opts *TagOptions) GetBool(key string) (bool, error) {
	return strconv.ParseBool(opts.Get(key))
}

func (opts *TagOptions) HasAny(keys ...string) bool {
	return slices.ContainsFunc(keys, opts.Has)
}

func (opts *TagOptions) GetAny(keys ...string) (string, bool) {
	for _, key := range keys {
		if opts.Has(key) {
			return opts.Get(key), true
		}
	}
	return "", false
}

func (opts *TagOptions) GetAll(keys ...string) []string {
	vv := []string{}
	for _, key := range keys {
		if opts.Has(key) {
			vv = append(vv, opts.Values[key]...)
		}
	}
	return vv
}
