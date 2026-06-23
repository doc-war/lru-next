package internal

import (
	"testing"
)

func TestLRUCoreAddGet(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("b", "2")

	v, ok := l.Get("a")
	if !ok || v != "1" {
		t.Fatalf("expected '1', got '%v'", v)
	}
	v, ok = l.Get("b")
	if !ok || v != "2" {
		t.Fatalf("expected '2', got '%v'", v)
	}
}

func TestLRUCoreGetNonExistent(t *testing.T) {
	l := newLRUCore[string](3)
	_, ok := l.Get("nope")
	if ok {
		t.Fatal("expected false for missing key")
	}
}

func TestLRUCoreEviction(t *testing.T) {
	l := newLRUCore[string](2)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Add("c", "3") // should evict "a"

	if _, ok := l.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if _, ok := l.Get("b"); !ok {
		t.Fatal("expected 'b' to exist")
	}
	if _, ok := l.Get("c"); !ok {
		t.Fatal("expected 'c' to exist")
	}
	if l.Len() != 2 {
		t.Fatalf("expected Len=2, got %d", l.Len())
	}
}

func TestLRUCoreEvictLRU(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Add("c", "3")
	l.Get("a")     // a → front, order: a, c, b
	l.Add("d", "4") // should evict "b" (tail)

	if _, ok := l.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted (LRU)")
	}
}

func TestLRUCoreUpdate(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("a", "42") // update

	v, ok := l.Get("a")
	if !ok || v != "42" {
		t.Fatalf("expected '42', got '%v'", v)
	}
	if l.Len() != 1 {
		t.Fatalf("expected Len=1, got %d", l.Len())
	}
}

func TestLRUCoreRemove(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Remove("a")

	if _, ok := l.Get("a"); ok {
		t.Fatal("expected 'a' to be removed")
	}
	if l.Len() != 1 {
		t.Fatalf("expected Len=1, got %d", l.Len())
	}
}

func TestLRUCoreRemoveNonExistent(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Remove("nope") // should not panic
	if l.Len() != 1 {
		t.Fatalf("expected Len=1, got %d", l.Len())
	}
}

func TestLRUCorePurge(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Purge()

	if l.Len() != 0 {
		t.Fatalf("expected Len=0 after purge, got %d", l.Len())
	}
	if _, ok := l.Get("a"); ok {
		t.Fatal("expected no values after purge")
	}
}

func TestLRUCoreAddAfterEviction(t *testing.T) {
	l := newLRUCore[string](2)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Add("c", "3") // evicts a
	l.Add("d", "4") // evicts b

	if _, ok := l.Get("c"); !ok {
		t.Fatal("expected 'c' to exist")
	}
	if _, ok := l.Get("d"); !ok {
		t.Fatal("expected 'd' to exist")
	}
	if _, ok := l.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if _, ok := l.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
}

func TestLRUCoreMinCapacity(t *testing.T) {
	l := newLRUCore[string](0) // should be clamped to 1
	l.Add("a", "1")
	l.Add("b", "2") // evicts a

	if _, ok := l.Get("a"); ok {
		t.Fatal("expected 'a' to be evicted (min capacity=1)")
	}
	if l.Len() != 1 {
		t.Fatalf("expected Len=1, got %d", l.Len())
	}
}

func TestLRUCoreMoveToFrontOnAddExisting(t *testing.T) {
	l := newLRUCore[string](3)
	l.Add("a", "1")
	l.Add("b", "2")
	l.Add("c", "3")
	// order: c(head), b, a(tail)
	l.Add("a", "10") // update + move a to front
	// order: a(head), c, b(tail)
	l.Add("d", "4") // should evict b (tail)

	if _, ok := l.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted (was tail)")
	}
}
