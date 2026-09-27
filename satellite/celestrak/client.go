// Package celestrak fetches satellite element sets from CelesTrak's GP API
// (https://celestrak.org/NORAD/elements/gp.php) as CCSDS OMM JSON, for a
// caller who wants the current set rather than supplying its own. It is the
// explicit fetcher behind the satellite packages: nothing is fetched unless a
// caller builds a Client and passes it as the element source.
//
// Element sets are cached by catalogue number only, never per observer or
// time, and every position and pass is propagated locally from the cached
// set. The network is hit rarely, and a caller never waits on it once a set
// is cached:
//
//   - The first call for a catalogue number fetches synchronously; callers
//     that arrive together share that one request.
//   - After that every call answers straight from the cache. Once the set is
//     older than the refresh age (3 days by default), one background refresh
//     starts, de-duplicated per catalogue number and never sooner than
//     MinRefetch (2 hours) after the last attempt, following CelesTrak's usage
//     guidance. A failed refresh keeps the old set, is logged at warn, and is
//     retried after MinRefetch.
//   - With NeverExpire the set is fetched once and never refreshed.
//
// The default cache is in-process and is lost on restart. The cache is
// pluggable (satellite.Cache), so a caller can back it with its own store,
// such as Redis, to keep sets across restarts and share them between
// replicas.
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
	// DefaultRefreshAge is how old a cached set gets before a background
	// refresh starts.
	DefaultRefreshAge = 3 * 24 * time.Hour
	// MinRefetch is the shortest interval between requests for one set, and
	// the floor for the refresh age.
	MinRefetch = 2 * time.Hour
	// DefaultTimeout bounds each request.
	DefaultTimeout = 15 * time.Second
)

// Client fetches element sets. Build it with New; it is safe for concurrent
// use and implements satellite.ElementSource.
type Client struct {
	http       *http.Client
	base       string
	timeout    time.Duration
	refreshAge time.Duration
	never      bool
	cache      satellite.Cache
	now        func() time.Time
	log        log.Logger

	mu       sync.Mutex
	attempts map[int]time.Time // last request per catalogue number, any outcome
	failures map[int]time.Time // last failed first fetch per catalogue number
	inFlight map[int]*call     // a request under way per catalogue number
}

// call is one request that callers can share.
type call struct {
	done chan struct{}
	res  Result
	err  error
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

// WithRefreshAge sets how old a cached set gets before a background refresh
// starts. It is never less than MinRefetch.
func WithRefreshAge(d time.Duration) Option {
	return func(cl *Client) {
		if d > 0 {
			cl.refreshAge = max(d, MinRefetch)
		}
	}
}

// NeverExpire fetches each set once and never refreshes it. Propagation from
// an old set drifts: fine for a few days, tens to hundreds of kilometres after
// about a week for a low orbit such as the ISS, and far off after a month or
// after a reboost. A set too old for SGP4 to propagate gives an error, never
// a wrong position.
func NeverExpire() Option {
	return func(cl *Client) { cl.never = true }
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
		http:       &http.Client{},
		base:       DefaultBaseURL,
		timeout:    DefaultTimeout,
		refreshAge: DefaultRefreshAge,
		now:        time.Now,
		attempts:   map[int]time.Time{},
		failures:   map[int]time.Time{},
		inFlight:   map[int]*call{},
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
