package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

type item struct {
	value      []byte
	expiration int64
}

type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]item

	// simple queue per key
	queues map[string][]string
	cond   *sync.Cond
}

func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{
		store:  make(map[string]item),
		queues: make(map[string][]string),
	}
	c.cond = sync.NewCond(&c.mu)

	// background cleanup
	go c.cleanupExpired()
	return c
}

// -------------------- KV --------------------
func (c *InMemoryCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	it, ok := c.store[key]
	c.mu.RUnlock()

	if !ok {
		return "", errors.New("not found")
	}

	if it.expiration > 0 && time.Now().UnixNano() > it.expiration {
		c.Delete(ctx, key)
		return "", errors.New("expired")
	}

	return string(it.value), nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	c.store[key] = item{
		value:      value,
		expiration: exp,
	}
	c.mu.Unlock()

	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
}

// simple pattern delete (prefix-based, not full Redis glob)
func (c *InMemoryCache) DeleteByPattern(ctx context.Context, pattern string) error {
	prefix := normalizePatternPrefix(pattern)
	if prefix == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
		}
	}
	return nil
}

func normalizePatternPrefix(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}

	// Support both "events:*" and direct prefixes like "tickets:".
	if strings.HasSuffix(pattern, "*") {
		return strings.TrimSuffix(pattern, "*")
	}

	return pattern
}

// -------------------- JSON --------------------

func (c *InMemoryCache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, data, ttl)
}

// -------------------- Queue --------------------
func (c *InMemoryCache) LPush(ctx context.Context, key string, value []byte) error {
	c.mu.Lock()
	c.queues[key] = append([]string{string(value)}, c.queues[key]...)
	c.mu.Unlock()

	c.cond.Signal()
	return nil
}

func (c *InMemoryCache) BRPop(ctx context.Context, timeout time.Duration, key string) (string, error) {
	deadline := time.Now().Add(timeout)

	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		if q := c.queues[key]; len(q) > 0 {
			val := q[len(q)-1]
			c.queues[key] = q[:len(q)-1]
			return val, nil
		}

		if timeout > 0 && time.Now().After(deadline) {
			return "", nil
		}

		c.cond.Wait()
	}
}

func (c *InMemoryCache) QueueLength(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return int64(len(c.queues[key])), nil
}

// -------------------- Cleanup --------------------

func (c *InMemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now().UnixNano()

		c.mu.Lock()
		for k, v := range c.store {
			if v.expiration > 0 && now > v.expiration {
				delete(c.store, k)
			}
		}
		c.mu.Unlock()
	}
}

// optional (for compatibility)
func (c *InMemoryCache) Close() error {
	return nil
}
