package ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Store interface {
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
	Reset(ctx context.Context, key string) error
}

type MemoryStore struct {
	mu    sync.Mutex
	data  map[string]*windowEntry
}

type windowEntry struct {
	count    int
	windowStart time.Time
	duration time.Duration
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]*windowEntry),
	}
}

func (s *MemoryStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[key]
	if !exists || time.Since(entry.windowStart) > entry.duration {
		s.data[key] = &windowEntry{
			count:       1,
			windowStart: time.Now(),
			duration:    window,
		}
		return 1, nil
	}

	entry.count++
	return entry.count, nil
}

func (s *MemoryStore) Reset(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

type Config struct {
	Enabled  bool
	MaxRequests int
	Window   time.Duration
}

type Limiter struct {
	store Store
	cfg   Config
}

func NewLimiter(store Store, cfg Config) *Limiter {
	return &Limiter{
		store: store,
		cfg:   cfg,
	}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		key := l.keyFromRequest(r)

		count, err := l.store.Increment(r.Context(), key, l.cfg.Window)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.cfg.MaxRequests))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, l.cfg.MaxRequests-count)))

		if count > l.cfg.MaxRequests {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.cfg.Window.Seconds())))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(l.cfg.Window).Unix(), 10))
			http.Error(w, `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) keyFromRequest(r *http.Request) string {
	ip := extractIP(r)
	token := ""
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = auth[7:]
		if len(token) > 16 {
			token = token[:16]
		}
	}

	if token != "" {
		return fmt.Sprintf("ratelimit:token:%s:%s", token, r.URL.Path)
	}

	return fmt.Sprintf("ratelimit:ip:%s:%s", ip, r.URL.Path)
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

type RedisStore struct {
	// Pending: Redis implementation
}

func NewRedisStore() *RedisStore {
	return &RedisStore{}
}

func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	return 0, fmt.Errorf("redis store not yet implemented, use memory store")
}

func (s *RedisStore) Reset(ctx context.Context, key string) error {
	return fmt.Errorf("redis store not yet implemented, use memory store")
}
