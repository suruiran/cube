package cmap

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

const (
	keyCount    = 1000000
	bucketCount = 64
)

func BenchmarkMapComparison(b *testing.B) {
	keys := make([]string, keyCount)
	for i := range keyCount {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	scenarios := []struct {
		name       string
		readWeight int
	}{
		{"Read90_Write10", 90},
		{"Read50_Write50", 50},
		{"Read10_Write90", 10},
	}

	for _, sc := range scenarios {
		b.Run(fmt.Sprintf("SyncMap_%s", sc.name), func(b *testing.B) {
			var sm sync.Map
			for _, k := range keys {
				sm.Store(k, "value")
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(42))
				for pb.Next() {
					key := keys[r.Intn(keyCount)]
					if r.Intn(100) < sc.readWeight {
						sm.Load(key)
					} else {
						sm.Store(key, "value")
					}
				}
			})
		})

		b.Run(fmt.Sprintf("CMap_%s", sc.name), func(b *testing.B) {
			cm := New[string, string](bucketCount)
			for _, k := range keys {
				cm.Set(k, "value")
			}
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(42))
				for pb.Next() {
					key := keys[r.Intn(keyCount)]
					if r.Intn(100) < sc.readWeight {
						cm.Get(key)
					} else {
						cm.Set(key, "value")
					}
				}
			})
		})
	}
}
