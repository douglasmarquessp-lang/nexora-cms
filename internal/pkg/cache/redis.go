package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type redisDriver struct {
	mu       sync.RWMutex
	host     string
	port     int
	password string
	entries  map[string]redisCacheEntry
}

type redisCacheEntry struct {
	data      []byte
	expiresAt time.Time
}

func NewRedisDriver(host string, port int, password string) Driver {
	return &redisDriver{
		host:     host,
		port:     port,
		password: password,
		entries:  make(map[string]redisCacheEntry),
	}
}

func (r *redisDriver) Get(ctx context.Context, key string) ([]byte, error) {
	r.mu.RLock()
	entry, ok := r.entries[key]
	r.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		r.mu.Lock()
		delete(r.entries, key)
		r.mu.Unlock()
		return nil, nil
	}
	return entry.data, nil
}

func (r *redisDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := redisCacheEntry{data: make([]byte, len(value))}
	copy(entry.data, value)
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	r.entries[key] = entry
	return nil
}

func (r *redisDriver) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)

	for k := range r.entries {
		if len(k) > len(key) && k[:len(key)] == key {
			delete(r.entries, k)
		}
	}

	return nil
}

func (r *redisDriver) Flush(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]redisCacheEntry)
	return nil
}

func (r *redisDriver) Exists(ctx context.Context, key string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.entries[key]
	return ok, nil
}

func (r *redisDriver) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := r.Get(ctx, key)
	if err != nil || data == nil {
		return false, nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache: failed to unmarshal: %w", err)
	}
	return true, nil
}

func (r *redisDriver) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: failed to marshal: %w", err)
	}
	return r.Set(ctx, key, data, ttl)
}

func (r *redisDriver) Ping(ctx context.Context) error {
	return nil
}

func (r *redisDriver) Close() error {
	r.entries = nil
	return nil
}
