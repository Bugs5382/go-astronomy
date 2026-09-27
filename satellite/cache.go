package satellite

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

// A small cache the element and ephemeris fetchers share, so a caller can
// plug in its own store (Redis, say) and share fetched data across restarts
// and replicas. The package itself depends on no store.

import (
	"context"
	"sync"
	"time"
)

// Cache stores byte values under string keys with an expiry. Get reports
// whether a live value is held; Set stores one that expires after expiry, or
// never for an expiry of zero. Implementations must be safe for concurrent
// use. An error from either means the store failed; the fetchers treat that
// as a cache miss and carry on.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error
}

// MemoryCache is the default Cache: an in-process map.
type MemoryCache struct {
	mu   sync.Mutex
	now  func() time.Time
	data map[string]memoryEntry
}

type memoryEntry struct {
	value   []byte
	expires time.Time // zero: never
}

var _ Cache = (*MemoryCache)(nil)

// CacheOption configures a MemoryCache.
type CacheOption func(*MemoryCache)

// WithCacheClock replaces the clock, for tests.
func WithCacheClock(now func() time.Time) CacheOption {
	return func(c *MemoryCache) {
		if now != nil {
			c.now = now
		}
	}
}

// NewMemoryCache returns an empty in-process cache.
func NewMemoryCache(opts ...CacheOption) *MemoryCache {
	c := &MemoryCache{now: time.Now, data: map[string]memoryEntry{}}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Get returns a copy of the live value under key.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok {
		return nil, false, nil
	}
	if !e.expires.IsZero() && !c.now().Before(e.expires) {
		delete(c.data, key)
		return nil, false, nil
	}
	return append([]byte(nil), e.value...), true, nil
}

// Set stores a copy of value under key.
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, expiry time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := memoryEntry{value: append([]byte(nil), value...)}
	if expiry > 0 {
		e.expires = c.now().Add(expiry)
	}
	c.data[key] = e
	return nil
}
