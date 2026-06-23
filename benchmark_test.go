package cache_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/doc-war/TTLCacheNext"

	gocache "github.com/patrickmn/go-cache"
	lru "github.com/hashicorp/golang-lru/v2"
)

// fixed key pool — avoids OOM when b.N is large
const poolSize = 10000

var stringKeys []string

func init() {
	stringKeys = make([]string, poolSize)
	for i := range stringKeys {
		stringKeys[i] = fmt.Sprintf("k%016d", i)
	}
}

// ---- TTLCacheNext benchmarks ----

func BenchmarkTTLCacheNext_ParallelSet(b *testing.B) {
	c, _ := cache.New[string](poolSize, time.Hour)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			c.GetOrLoad(k, func(s string) (string, error) {
				return "v", nil
			})
		}
	})
}

func BenchmarkTTLCacheNext_ParallelGet(b *testing.B) {
	c, _ := cache.New[string](poolSize, time.Hour)
	for _, k := range stringKeys {
		c.GetOrLoad(k, func(s string) (string, error) {
			return "v", nil
		})
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			c.GetOrLoad(k, func(s string) (string, error) {
				return "v", nil
			})
		}
	})
}

func BenchmarkTTLCacheNext_Mixed(b *testing.B) {
	c, _ := cache.New[string](poolSize, time.Hour)
	for _, k := range stringKeys {
		c.GetOrLoad(k, func(s string) (string, error) {
			return "v", nil
		})
	}
	var mu sync.Mutex
	writeIdx := poolSize
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				mu.Lock()
				k := fmt.Sprintf("k%016d", writeIdx)
				writeIdx++
				mu.Unlock()
				c.GetOrLoad(k, func(s string) (string, error) {
					return "v", nil
				})
			} else {
				k := stringKeys[rand.Intn(poolSize)]
				c.GetOrLoad(k, func(s string) (string, error) {
					return "v", nil
				})
			}
		}
	})
}

// ---- golang-lru benchmarks ----

func BenchmarkGolangLRU_ParallelSet(b *testing.B) {
	lc, _ := lru.New[string, string](poolSize)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			lc.Add(k, "v")
		}
	})
}

func BenchmarkGolangLRU_ParallelGet(b *testing.B) {
	lc, _ := lru.New[string, string](poolSize)
	for _, k := range stringKeys {
		lc.Add(k, "v")
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			lc.Get(k)
		}
	})
}

func BenchmarkGolangLRU_Mixed(b *testing.B) {
	lc, _ := lru.New[string, string](poolSize)
	for _, k := range stringKeys {
		lc.Add(k, "v")
	}
	var mu sync.Mutex
	writeIdx := poolSize
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				mu.Lock()
				k := fmt.Sprintf("k%016d", writeIdx)
				writeIdx++
				mu.Unlock()
				lc.Add(k, "v")
			} else {
				k := stringKeys[rand.Intn(poolSize)]
				lc.Get(k)
			}
		}
	})
}

// ---- go-cache benchmarks ----

func BenchmarkGoCache_ParallelSet(b *testing.B) {
	gc := gocache.New(time.Hour, time.Hour)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			gc.Set(k, "v", gocache.DefaultExpiration)
		}
	})
}

func BenchmarkGoCache_ParallelGet(b *testing.B) {
	gc := gocache.New(time.Hour, time.Hour)
	for _, k := range stringKeys {
		gc.Set(k, "v", gocache.DefaultExpiration)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			k := stringKeys[rand.Intn(poolSize)]
			gc.Get(k)
		}
	})
}

func BenchmarkGoCache_Mixed(b *testing.B) {
	gc := gocache.New(time.Hour, time.Hour)
	for _, k := range stringKeys {
		gc.Set(k, "v", gocache.DefaultExpiration)
	}
	var mu sync.Mutex
	writeIdx := poolSize
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				mu.Lock()
				k := fmt.Sprintf("k%016d", writeIdx)
				writeIdx++
				mu.Unlock()
				gc.Set(k, "v", gocache.DefaultExpiration)
			} else {
				k := stringKeys[rand.Intn(poolSize)]
				gc.Get(k)
			}
		}
	})
}
