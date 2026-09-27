// Package openmeteo looks elevations up from the Open-Meteo elevation API
// (https://open-meteo.com/en/docs/elevation-api), which serves the Copernicus
// DEM at 90 m resolution with no key. It implements elevation.Lookup with the
// standard library's net/http only.
//
// The client keeps every value it fetches in an in-process map with no expiry,
// keyed by the coordinate rounded to a grid (0.01 degree by default, about
// 1.1 km), because the ground does not move: one process asks once per cell.
// The query is made at the cell's rounded coordinate, so every point in a cell
// shares one value. A durable cache across processes is the consumer's; Cached
// and Store let it seed or read the map.
//
// The client is silent unless given a go-log Logger with WithLogger, in which
// case it logs each lookup, cache hit, request, and failure.
package openmeteo

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	log "github.com/Bugs5382/go-log"

	"github.com/Bugs5382/go-astronomy/elevation"
)

// DefaultBaseURL is the Open-Meteo elevation endpoint.
const DefaultBaseURL = "https://api.open-meteo.com/v1/elevation"

// DefaultRound is the default cache grid, in degrees.
const DefaultRound = 0.01

// defaultTimeout bounds a request made with the default HTTP client.
const defaultTimeout = 10 * time.Second

// maxBody bounds how much of a response is read. A single elevation is a few
// dozen bytes.
const maxBody = 64 << 10

// Client is an elevation.Lookup backed by the Open-Meteo elevation API. Build
// it with New. It is safe for concurrent use; two concurrent misses on the same
// cell may both reach the service, and both store the same value.
type Client struct {
	http  *http.Client
	base  string
	round float64
	log   log.Logger

	mu    sync.RWMutex
	cache map[cell]float64
}

var _ elevation.Lookup = (*Client)(nil)

// cell is a cache key: a coordinate as whole multiples of the grid step.
type cell struct{ lat, lng int64 }

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for requests. The default has a
// ten-second timeout.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithBaseURL sets the endpoint, for a self-hosted Open-Meteo or a test
// server. The default is DefaultBaseURL.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		if u != "" {
			c.base = u
		}
	}
}

// WithRound sets the cache grid in degrees. A non-positive or non-finite value
// is ignored and DefaultRound is kept.
func WithRound(deg float64) Option {
	return func(c *Client) {
		if deg > 0 && !math.IsInf(deg, 0) {
			c.round = deg
		}
	}
}

// WithLogger makes the client log its lookups through l: cache hits and
// requests at debug level, failures at warn. Coordinates are logged rounded
// to the cache grid. Without it the client logs nothing.
func WithLogger(l log.Logger) Option {
	return func(c *Client) { c.log = l }
}

// New returns a Client with the given options applied.
func New(opts ...Option) *Client {
	c := &Client{
		http:  &http.Client{Timeout: defaultTimeout},
		base:  DefaultBaseURL,
		round: DefaultRound,
		cache: map[cell]float64{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// key returns the cache cell of a coordinate and the coordinate of that
// cell, which is what gets queried.
func (c *Client) key(lat, lng float64) (cell, float64, float64) {
	k := cell{int64(math.Round(lat / c.round)), int64(math.Round(lng / c.round))}
	return k, float64(k.lat) * c.round, float64(k.lng) * c.round
}

// Elevation returns the height above sea level, in metres, of the ground at
// the coordinate, from the cache or from Open-Meteo. Errors are go-apperr
// coded: elevation.ErrInvalidLatitude or ErrInvalidLongitude for a coordinate
// out of range, and elevation.ErrLookupFailed for anything the service or the
// network got wrong, wrapping the underlying cause.
func (c *Client) Elevation(ctx context.Context, lat, lng float64) (float64, error) {
	if err := elevation.ValidateCoordinate(lat, lng); err != nil {
		return 0, err
	}
	k, qlat, qlng := c.key(lat, lng)
	c.mu.RLock()
	h, ok := c.cache[k]
	c.mu.RUnlock()
	if ok {
		c.debug("elevation cache hit", log.F("lat", qlat), log.F("lng", qlng), log.F("elevation_m", h))
		return h, nil
	}
	c.debug("elevation cache miss", log.F("lat", qlat), log.F("lng", qlng))

	h, err := c.fetch(ctx, qlat, qlng)
	if err != nil {
		if c.log != nil {
			c.log.Warn("elevation lookup failed", log.F("lat", qlat), log.F("lng", qlng), log.F("error", err.Error()))
		}
		return 0, elevation.LookupFailed(err)
	}
	c.Store(lat, lng, h)
	return h, nil
}

// Cached returns the cached height for the coordinate's cell and whether one
// is held, without any request.
func (c *Client) Cached(lat, lng float64) (float64, bool) {
	k, _, _ := c.key(lat, lng)
	c.mu.RLock()
	defer c.mu.RUnlock()
	h, ok := c.cache[k]
	return h, ok
}

// Store records a height for the coordinate's cell, for a consumer seeding
// the cache from its own durable store.
func (c *Client) Store(lat, lng, elevationM float64) {
	k, _, _ := c.key(lat, lng)
	c.mu.Lock()
	c.cache[k] = elevationM
	c.mu.Unlock()
}

// response is the Open-Meteo elevation reply: one height per requested
// coordinate, or an error flag and reason.
type response struct {
	Elevation []float64 `json:"elevation"`
	Error     bool      `json:"error"`
	Reason    string    `json:"reason"`
}

// errService is the cause when Open-Meteo answers with an error.
var errService = errors.New("open-meteo error")

// fetch asks Open-Meteo for one coordinate.
func (c *Client) fetch(ctx context.Context, lat, lng float64) (float64, error) {
	q := url.Values{}
	q.Set("latitude", coord(lat))
	q.Set("longitude", coord(lng))
	target := c.base + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-astronomy")

	start := time.Now()
	resp, err := c.http.Do(req) // #nosec G107 G704 -- the URL is the configured endpoint plus two numbers.
	if err != nil {
		return 0, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	c.debug("elevation request", log.F("url", target), log.F("status", resp.StatusCode),
		log.F("duration_ms", time.Since(start).Milliseconds()))
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, fmt.Errorf("status %d: decode response: %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || r.Error {
		return 0, fmt.Errorf("%w: status %d: %s", errService, resp.StatusCode, r.Reason)
	}
	if len(r.Elevation) != 1 {
		return 0, fmt.Errorf("got %d elevations, want 1", len(r.Elevation))
	}
	h := r.Elevation[0]
	if math.IsNaN(h) || math.IsInf(h, 0) {
		return 0, fmt.Errorf("elevation %v is not a number", h)
	}
	c.debug("elevation fetched", log.F("lat", lat), log.F("lng", lng), log.F("elevation_m", h))
	return h, nil
}

// coord formats a cell coordinate to the microdegree, which drops the float
// noise of multiplying the grid step back out (-104.99000000000001).
func coord(v float64) string {
	return strconv.FormatFloat(math.Round(v*1e6)/1e6, 'f', -1, 64)
}

func (c *Client) debug(msg string, fields ...log.Field) {
	if c.log != nil {
		c.log.Debug(msg, fields...)
	}
}
