package cache

import "testing"

func TestShardedCacheSetAndGet(t *testing.T) {
	// Create a sharded cache with 4 shards and a capacity of 2 per shard.
	cache := NewShardedCache(WithShardCount(4), WithShardCapacity(2))

	cache.Set("foo", "bar")
	value, err := cache.Get("foo")
	if err != nil {
		t.Fatalf("expected key 'foo' to exist, got error: %v", err)
	}
	if value != "bar" {
		t.Fatalf("expected value 'bar', got %q", value)
	}
}

func TestShardedCacheEviction(t *testing.T) {
	// Use one shard (to test eviction in isolation) with capacity 2.
	cache := NewShardedCache(WithShardCount(1), WithShardCapacity(2))

	cache.Set("a", "1")
	cache.Set("b", "2")

	// Access "a" to mark it as recently used.
	if _, err := cache.Get("a"); err != nil {
		t.Fatal("expected key 'a' to exist")
	}

	// Inserting a third key should evict the least recently used ("b").
	cache.Set("c", "3")

	// "b" should be evicted.
	if _, err := cache.Get("b"); err == nil {
		t.Fatal("expected key 'b' to be evicted")
	}

	// "a" and "c" should still be present.
	if v, err := cache.Get("a"); err != nil || v != "1" {
		t.Fatal("expected key 'a' to exist with value '1'")
	}
	if v, err := cache.Get("c"); err != nil || v != "3" {
		t.Fatal("expected key 'c' to exist with value '3'")
	}
}

func TestShardedCacheDelete(t *testing.T) {
	cache := NewShardedCache(WithShardCount(4), WithShardCapacity(2))
	cache.Set("test", "value")

	// Verify key exists.
	if v, err := cache.Get("test"); err != nil || v != "value" {
		t.Fatalf("expected key 'test' to exist, got error: %v", err)
	}

	// Delete and verify removal.
	cache.Delete("test")
	if _, err := cache.Get("test"); err == nil {
		t.Fatal("expected key 'test' to be deleted")
	}
}
