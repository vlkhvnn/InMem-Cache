package cache

import (
	"container/list"
	"errors"
	"hash/fnv"
	"sync"
)

// entry represents a key-value pair stored in the cache.
type entry struct {
	key   string
	value string
}

// Shard represents a partition of the cache.
// It holds its own data map, LRU list for eviction, and a mutex.
type Shard struct {
	mu       sync.Mutex
	data     map[string]*list.Element
	lru      *list.List
	capacity int
}

// newShard creates a new shard with a given capacity.
// If capacity <= 0, the shard will be treated as having unlimited capacity.
func newShard(capacity int) *Shard {
	return &Shard{
		data:     make(map[string]*list.Element),
		lru:      list.New(),
		capacity: capacity,
	}
}

// set inserts or updates a key-value pair in the shard.
// If the key exists, it updates its value and moves it to the front of the LRU list.
// If the shard is at capacity, it evicts the least recently used item.
func (s *Shard) set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If key exists, update the value and move to front.
	if elem, ok := s.data[key]; ok {
		elem.Value.(*entry).value = value
		s.lru.MoveToFront(elem)
		return
	}

	// If capacity is set and reached, evict the least recently used entry.
	if s.capacity > 0 && s.lru.Len() >= s.capacity {
		s.evict()
	}

	// Insert new entry at the front of the LRU list.
	ent := &entry{key: key, value: value}
	elem := s.lru.PushFront(ent)
	s.data[key] = elem
}

// get retrieves a key's value from the shard and updates its position in the LRU list.
func (s *Shard) get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.data[key]; ok {
		s.lru.MoveToFront(elem)
		return elem.Value.(*entry).value, nil
	}
	return "", errors.New("key not found")
}

// delete removes a key from the shard.
func (s *Shard) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.data[key]; ok {
		s.lru.Remove(elem)
		delete(s.data, key)
	}
}

// evict removes the least recently used item from the shard.
func (s *Shard) evict() {
	elem := s.lru.Back()
	if elem == nil {
		return
	}
	ent := elem.Value.(*entry)
	delete(s.data, ent.key)
	s.lru.Remove(elem)
}

// ShardedCache represents a thread-safe in-memory cache that partitions keys into shards.
type ShardedCache struct {
	shards        []*Shard
	shardCount    int
	shardCapacity int
}

// Option represents a functional option for configuring the ShardedCache.
type Option func(*ShardedCache)

// WithShardCount sets the number of shards in the cache.
func WithShardCount(n int) Option {
	return func(sc *ShardedCache) {
		if n > 0 {
			sc.shardCount = n
		}
	}
}

// WithShardCapacity sets the capacity for each shard.
func WithShardCapacity(cap int) Option {
	return func(sc *ShardedCache) {
		if cap > 0 {
			sc.shardCapacity = cap
		}
	}
}

// NewShardedCache creates a new ShardedCache instance with the provided options.
// Defaults: 16 shards, 100 items per shard.
func NewShardedCache(opts ...Option) *ShardedCache {
	sc := &ShardedCache{
		shardCount:    16,
		shardCapacity: 100,
	}
	// Apply options.
	for _, opt := range opts {
		opt(sc)
	}
	// Initialize shards.
	sc.shards = make([]*Shard, sc.shardCount)
	for i := 0; i < sc.shardCount; i++ {
		sc.shards[i] = newShard(sc.shardCapacity)
	}
	return sc
}

// getShard selects a shard based on the key's hash.
func (sc *ShardedCache) getShard(key string) *Shard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	idx := hash.Sum32() % uint32(sc.shardCount)
	return sc.shards[idx]
}

// Set inserts or updates the key-value pair in the appropriate shard.
func (sc *ShardedCache) Set(key, value string) {
	shard := sc.getShard(key)
	shard.set(key, value)
}

// Get retrieves the value for a key from the appropriate shard.
func (sc *ShardedCache) Get(key string) (string, error) {
	shard := sc.getShard(key)
	return shard.get(key)
}

// Delete removes the key from the appropriate shard.
func (sc *ShardedCache) Delete(key string) {
	shard := sc.getShard(key)
	shard.delete(key)
}
