package cache

import "testing"

func TestCacheSetAndGet(t *testing.T) {
	c := NewCache()
	c.Set("hello", "world")

	value, err := c.Get("hello")
	if err != nil {
		t.Fatalf("expected key 'hello' to exist, got error: %v", err)
	}

	if value != "world" {
		t.Fatalf("expected value 'world', got %q", value)
	}
}

func TestCacheGetNonExistent(t *testing.T) {
	c := NewCache()
	_, err := c.Get("nonexistent")
	if err == nil {
		t.Fatal("expected an error for a nonexistent key")
	}
}

func TestCacheDelete(t *testing.T) {
	c := NewCache()
	c.Set("key", "value")

	c.Delete("key")

	_, err := c.Get("key")
	if err == nil {
		t.Fatal("expected an error after deleting the key")
	}
}
