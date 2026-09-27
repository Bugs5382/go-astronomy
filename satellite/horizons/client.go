package horizons

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
	// DefaultBaseURL is the Horizons API endpoint.
	DefaultBaseURL = "https://ssd.jpl.nasa.gov/api/horizons.api"
	// DefaultWindow is the span one request covers.
	DefaultWindow = 30 * 24 * time.Hour
	// DefaultStep is the table step.
	DefaultStep = time.Hour
	// DefaultTimeout bounds each request; a 30-day hourly table is a few
	// seconds of work for Horizons.
	DefaultTimeout = 60 * time.Second
	// pad is the number of extra steps fetched on each side of a window, so
	// the interpolation has points on both sides of any instant inside it.
	pad = 4
)

// Client fetches and interpolates Horizons ephemerides. Build it with New; it
// is safe for concurrent use.
type Client struct {
	http    *http.Client
	base    string
	timeout time.Duration
	window  time.Duration
	step    time.Duration
	cache   satellite.Cache
	log     log.Logger
}

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

// WithBaseURL sets the endpoint, for a test server.
func WithBaseURL(u string) Option {
	return func(cl *Client) {
		if u != "" {
			cl.base = u
		}
	}
}

// WithTimeout sets how long one request may take.
func WithTimeout(d time.Duration) Option {
	return func(cl *Client) {
		if d > 0 {
			cl.timeout = d
		}
	}
}

// WithWindow sets the span one request covers. Longer windows mean fewer
// requests and larger cached tables; the default is 30 days.
func WithWindow(d time.Duration) Option {
	return func(cl *Client) {
		if d > 0 {
			cl.window = d
		}
	}
}

// WithStep sets the table step, a whole number of minutes. Shorter steps cost
// larger tables for accuracy an L2 orbit does not need; the default is one
// hour.
func WithStep(d time.Duration) Option {
	return func(cl *Client) {
		if d >= time.Minute {
			cl.step = d.Truncate(time.Minute)
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
		window:  DefaultWindow,
		step:    DefaultStep,
	}
	for _, o := range opts {
		o(cl)
	}
	if cl.cache == nil {
		cl.cache = satellite.NewMemoryCache()
	}
	if cl.log == nil {
		cl.log = defaultLogger()
	}
	return cl
}
