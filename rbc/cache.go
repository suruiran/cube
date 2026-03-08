package rbc

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

type _StaticCache struct {
	items sync.Map
	sf    singleflight.Group

	runnings sync.Map
}

var (
	errReEntered = fmt.Errorf("cube.rbc: key is re-entered")
)

func (gc *_StaticCache) GetOrCompute(key string, compute func() (any, error)) (any, error) {
	if val, ok := gc.items.Load(key); ok {
		return val, nil
	}
	_, ok := gc.runnings.Load(key)
	if ok {
		return nil, errReEntered
	}

	val, err, _ := gc.sf.Do(key, func() (any, error) {
		_, loaded := gc.runnings.LoadOrStore(key, struct{}{})
		if loaded {
			return nil, errReEntered
		}
		defer gc.runnings.Delete(key)

		if val, ok := gc.items.Load(key); ok {
			return val, nil
		}
		_val, err := compute()
		if err != nil {
			return nil, err
		}
		gc.items.Store(key, _val)
		return _val, nil
	})
	return val, err
}
