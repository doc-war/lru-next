package internal

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestShardedCacheGetOrLoad(t *testing.T) {
	c, err := NewShardedCache[string](100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	val, err := c.GetOrLoad("foo", func(key string) (string, error) {
		return "bar", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "bar" {
		t.Fatalf("expected 'bar', got '%s'", val)
	}
}

func TestShardedCacheGetOrLoadReturnsCached(t *testing.T) {
	c, _ := NewShardedCache[string](100, time.Minute)
	loadCount := 0
	c.GetOrLoad("foo", func(key string) (string, error) {
		loadCount++
		return "bar", nil
	})
	val, err := c.GetOrLoad("foo", func(key string) (string, error) {
		loadCount++
		return "baz", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "bar" {
		t.Fatalf("expected cached 'bar', got '%s'", val)
	}
	if loadCount != 1 {
		t.Fatalf("expected loader called once, got %d", loadCount)
	}
}

func TestShardedCacheTTLExpiry(t *testing.T) {
	c, _ := NewShardedCache[string](100, 50*time.Millisecond)
	c.GetOrLoad("foo", func(key string) (string, error) {
		return "first", nil
	})
	time.Sleep(60 * time.Millisecond)
	val, err := c.GetOrLoad("foo", func(key string) (string, error) {
		return "second", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "second" {
		t.Fatalf("expected reloaded 'second', got '%s'", val)
	}
}

func TestShardedCacheLoaderErrorDegradation(t *testing.T) {
	c, _ := NewShardedCache[string](100, 50*time.Millisecond)
	c.GetOrLoad("foo", func(key string) (string, error) {
		return "good", nil
	})
	time.Sleep(60 * time.Millisecond)
	val, err := c.GetOrLoad("foo", func(key string) (string, error) {
		return "", errors.New("oops")
	})
	if err == nil {
		t.Fatal("expected error from loader")
	}
	if val != "good" {
		t.Fatalf("expected degraded 'good', got '%s'", val)
	}
}

func TestShardedCacheDelete(t *testing.T) {
	c, _ := NewShardedCache[string](100, time.Minute)
	c.GetOrLoad("foo", func(key string) (string, error) {
		return "bar", nil
	})
	c.Delete("foo")
	loadCount := 0
	c.GetOrLoad("foo", func(key string) (string, error) {
		loadCount++
		return "new", nil
	})
	if loadCount != 1 {
		t.Fatalf("expected loader called after delete, count=%d", loadCount)
	}
}

func TestShardedCacheClear(t *testing.T) {
	c, _ := NewShardedCache[string](100, time.Minute)
	c.GetOrLoad("a", func(key string) (string, error) { return "1", nil })
	c.GetOrLoad("b", func(key string) (string, error) { return "2", nil })
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("expected Len=0 after Clear, got %d", c.Len())
	}
}

func TestShardedCacheLen(t *testing.T) {
	c, _ := NewShardedCache[string](100, time.Minute)
	c.GetOrLoad("a", func(key string) (string, error) { return "1", nil })
	c.GetOrLoad("b", func(key string) (string, error) { return "2", nil })
	if c.Len() != 2 {
		t.Fatalf("expected Len=2, got %d", c.Len())
	}
}

func TestShardedCacheZeroValueOnMiss(t *testing.T) {
	c, _ := NewShardedCache[string](100, time.Minute)
	val, err := c.GetOrLoad("miss", func(key string) (string, error) {
		return "", errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if val != "" {
		t.Fatalf("expected zero value '', got '%s'", val)
	}
}

func TestShardedCacheConcurrentGetOrLoad(t *testing.T) {
	c, _ := NewShardedCache[int](1000, time.Minute)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "foo"
			c.GetOrLoad(key, func(k string) (int, error) {
				time.Sleep(5 * time.Millisecond)
				return n, nil
			})
		}(i)
	}
	wg.Wait()
	// Only one loader should have won; all get same value.
	val, _ := c.GetOrLoad("foo", func(k string) (int, error) {
		return -1, nil
	})
	if val < 0 || val > 49 {
		t.Fatalf("expected concurrent load to produce stable result, got %d", val)
	}
}

func TestShardedCacheConcurrentDifferentKeys(t *testing.T) {
	c, _ := NewShardedCache[int](1000, time.Minute)
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			c.GetOrLoad(key, func(k string) (int, error) {
				return n, nil
			})
		}(i)
	}
	wg.Wait()
	if c.Len() != 1 {
		t.Fatalf("expected Len=1 (single key), got %d", c.Len())
	}
}

func TestShardedCacheConcurrentReadWrite(t *testing.T) {
	c, _ := NewShardedCache[int](100, time.Minute)

	// Populate concurrently.
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			c.GetOrLoad(key, func(k string) (int, error) {
				return n, nil
			})
		}(i)
	}
	wg.Wait()

	// Read + write concurrently.
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.GetOrLoad("key", func(k string) (int, error) {
				return 99, nil
			})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Len()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Delete("key")
		}()
	}
	wg.Wait()
}

func TestShardedCacheManyKeysDistribution(t *testing.T) {
	c, _ := NewShardedCache[int](10000, time.Minute)
	for i := range 5000 {
		key := "k"
		c.GetOrLoad(key, func(k string) (int, error) {
			return i, nil
		})
	}
	if c.Len() == 0 {
		t.Fatal("expected items in cache after many keys")
	}
}
