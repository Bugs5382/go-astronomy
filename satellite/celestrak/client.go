// Package celestrak fetches satellite element sets from CelesTrak's GP API
// (https://celestrak.org/NORAD/elements/gp.php) as CCSDS OMM JSON, for a
// caller who wants the current set rather than supplying its own. It is the
// explicit fetcher behind the satellite packages: nothing is fetched unless a
// caller builds a Client and passes it as the element source.
//
// Element sets change a few times a day at most, so the network is hit
// rarely: each set is cached per catalogue number for a TTL of 24 hours by
// default, and never refetched sooner than every 2 hours, following
// CelesTrak's usage guidance (a set is updated a few times a day; download it
// once and reuse it). Every position and pass is then propagated locally from
// the cached set. The cache is pluggable (satellite.Cache), so replicas can
// share one store across restarts.
//
// A failed fetch returns an error. When a set was cached, it is returned too,
// marked stale, with an error wrapping satellite.ErrStaleElements, and the
// fetch is not retried for the minimum refetch interval.
package celestrak

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

import (
	"net/http"
	"sync"
	"time"

	log "github.com/Bugs5382/go-log"

	"github.com/Bugs5382/go-astronomy/satellite"
)

const (
	// DefaultBaseURL is CelesTrak's GP endpoint.
	DefaultBaseURL = "https://celestrak.org/NORAD/elements/gp.php"
	// DefaultTTL is how long a fetched set is used before it is refreshed.
	DefaultTTL = 24 * time.Hour
	// MinRefetch is the shortest interval between fetches of one set, and
	// the floor for the TTL.
	MinRefetch = 2 * time.Hour
	// DefaultTimeout bounds each request.
	DefaultTimeout = 15 * time.Second
	// retention keeps a set in the cache long after its TTL, so it can still
	// be returned as stale when a refresh fails.
	retention = 14 * 24 * time.Hour
)

// Client fetches element sets. Build it with New; it is safe for concurrent
// use and implements satellite.ElementSource.
type Client struct {
	http    *http.Client
	base    string
	timeout time.Duration
	ttl     time.Duration
	cache   satellite.Cache
	now     func() time.Time
	log     log.Logger

	mu     sync.Mutex
	failed map[int]time.Time // last failed refresh per catalogue number
}

var _ satellite.ElementSource = (*Client)(nil)

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client for requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		if c != nil {
			cl.http = c
		}
	}
}

// WithBaseURL sets the endpoint, for a mirror or a test server.
func WithBaseURL(u string) Option {
	return func(cl *Client) {
		if u != "" {
			cl.base = u
		}
	}
}

// WithTimeout sets how long one request may take. A non-positive value keeps
// DefaultTimeout.
func WithTimeout(d time.Duration) Option {
	return func(cl *Client) {
		if d > 0 {
			cl.timeout = d
		}
	}
}

// WithTTL sets how long a fetched set is used before it is refreshed. It is
// never less than MinRefetch.
func WithTTL(d time.Duration) Option {
	return func(cl *Client) {
		if d > 0 {
			cl.ttl = max(d, MinRefetch)
		}
	}
}

// WithCache sets the cache, for example a Redis-backed satellite.Cache shared
// by replicas. The default is an in-process satellite.MemoryCache.
func WithCache(c satellite.Cache) Option {
	return func(cl *Client) {
		if c != nil {
			cl.cache = c
		}
	}
}

// WithClock replaces the clock, for tests.
func WithClock(now func() time.Time) Option {
	return func(cl *Client) {
		if now != nil {
			cl.now = now
		}
	}
}

// WithLogger replaces the go-log logger.
func WithLogger(l log.Logger) Option {
	return func(cl *Client) {
		if l != nil {
			cl.log = l
		}
	}
}

var defaultLogger = sync.OnceValue(func() log.Logger { return log.NewLogger("go-astronomy") })

// New returns a Client with the options applied.
func New(opts ...Option) *Client {
	cl := &Client{
		http:    &http.Client{},
		base:    DefaultBaseURL,
		timeout: DefaultTimeout,
		ttl:     DefaultTTL,
		now:     time.Now,
		failed:  map[int]time.Time{},
	}
	for _, o := range opts {
		o(cl)
	}
	if cl.cache == nil {
		cl.cache = satellite.NewMemoryCache(satellite.WithCacheClock(cl.now))
	}
	if cl.log == nil {
		cl.log = defaultLogger()
	}
	return cl
}
