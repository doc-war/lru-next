package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCacheNew(t *testing.T) {
	c, err := New[string](100, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestCacheGetOrLoad(t *testing.T) {
	c, _ := New[string](100, time.Minute)
	val, err := c.GetOrLoad("hello", func(key string) (string, error) {
		return "world", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "world" {
		t.Fatalf("expected 'world', got '%s'", val)
	}
}

func TestCacheDelete(t *testing.T) {
	c, _ := New[string](100, time.Minute)
	c.GetOrLoad("a", func(key string) (string, error) { return "1", nil })
	c.Delete("a")
	count := 0
	c.GetOrLoad("a", func(key string) (string, error) {
		count++
		return "2", nil
	})
	if count != 1 {
		t.Fatalf("expected loader called after delete, count=%d", count)
	}
}

func TestCacheClear(t *testing.T) {
	c, _ := New[string](100, time.Minute)
	c.GetOrLoad("a", func(key string) (string, error) { return "1", nil })
	c.GetOrLoad("b", func(key string) (string, error) { return "2", nil })
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("expected Len=0 after Clear, got %d", c.Len())
	}
}

func TestCacheLen(t *testing.T) {
	c, _ := New[string](100, time.Minute)
	c.GetOrLoad("a", func(key string) (string, error) { return "1", nil })
	c.GetOrLoad("b", func(key string) (string, error) { return "2", nil })
	if c.Len() != 2 {
		t.Fatalf("expected Len=2, got %d", c.Len())
	}
}

func TestCacheConcurrent(t *testing.T) {
	c, _ := New[int](1000, time.Minute)
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.GetOrLoad("shared", func(k string) (int, error) {
				time.Sleep(2 * time.Millisecond)
				return 42, nil
			})
		}()
	}
	wg.Wait()
}
