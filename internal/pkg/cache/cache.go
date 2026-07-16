package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

type Driver interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Flush(ctx context.Context) error
	Exists(ctx context.Context, key string) (bool, error)
}

type memCacheEntry struct {
	data    []byte
	expires time.Time
}

type memoryCache struct {
	store sync.Map
}

func newMemoryCache() *memoryCache {
	return &memoryCache{}
}

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, error) {
	val, ok := c.store.Load(key)
	if !ok {
		return nil, nil
	}
	entry, ok := val.(memCacheEntry)
	if !ok {
		return nil, nil
	}
	if !entry.expires.IsZero() && time.Now().After(entry.expires) {
		c.store.Delete(key)
		return nil, nil
	}
	return entry.data, nil
}

func (c *memoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	entry := memCacheEntry{data: value}
	if ttl > 0 {
		entry.expires = time.Now().Add(ttl)
	}
	c.store.Store(key, entry)
	return nil
}

func (c *memoryCache) Delete(_ context.Context, key string) error {
	c.store.Delete(key)
	return nil
}

func (c *memoryCache) Flush(_ context.Context) error {
	c.store.Range(func(key, _ interface{}) bool {
		c.store.Delete(key)
		return true
	})
	return nil
}

func (c *memoryCache) Exists(_ context.Context, key string) (bool, error) {
	_, ok := c.store.Load(key)
	return ok, nil
}

type Cache struct {
	redis Driver
	mem   Driver
}

func New(memOnly bool) *Cache {
	return &Cache{
		mem: newMemoryCache(),
	}
}

func NewWithRedis(mem Driver, redis Driver) *Cache {
	return &Cache{
		redis: redis,
		mem:   mem,
	}
}

func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool) {
	if c.redis != nil {
		data, err := c.redis.Get(ctx, key)
		if err == nil && data != nil {
			var v interface{}
			if json.Unmarshal(data, &v) == nil {
				return v, true
			}
			return data, true
		}
	}

	data, err := c.mem.Get(ctx, key)
	if err != nil || data == nil {
		return nil, false
	}
	var v interface{}
	if json.Unmarshal(data, &v) == nil {
		return v, true
	}
	return data, true
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if c.redis != nil {
		if err := c.redis.Set(ctx, key, data, ttl); err != nil {
			slog.Warn("redis set failed", "key", key, "error", err)
		}
	}
	return c.mem.Set(ctx, key, data, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if c.redis != nil {
		if err := c.redis.Delete(ctx, key); err != nil {
			slog.Warn("redis delete failed", "key", key, "error", err)
		}
	}
	return c.mem.Delete(ctx, key)
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c.redis != nil {
		data, err := c.redis.Get(ctx, key)
		if err == nil && data != nil {
			if err := json.Unmarshal(data, dest); err == nil {
				return true, nil
			}
		}
	}

	data, err := c.mem.Get(ctx, key)
	if err != nil || data == nil {
		return false, nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if c.redis != nil {
		if err := c.redis.Set(ctx, key, data, ttl); err != nil {
			slog.Warn("redis setjson failed", "key", key, "error", err)
		}
	}
	return c.mem.Set(ctx, key, data, ttl)
}
