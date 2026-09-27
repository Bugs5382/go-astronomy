// Package openmeteo looks an observer's height up from the Open-Meteo
// elevation API (https://open-meteo.com/en/docs/elevation-api), which is
// worldwide, free, and needs no key, and serves the Copernicus DEM at 90 m. It
// implements astronomy.ElevationResolver with the standard library's net/http.
//
// It is the only package in the module that reaches the network, and only
// when asked: the calculation packages never import it. A lookup that fails
// in any way (a timeout, a non-200 reply, bad JSON, no value) falls back to
// sea level and says so with astronomy.SourceSeaLevel, never with an error;
// the error is only for a cancelled or expired context.
//
// Heights never change, so each is kept in an in-process map with no expiry,
// keyed by the coordinate rounded to a grid (0.01 degree by default, about
// 1.1 km), and the query is made at the cell's rounded coordinate so every
// point in a cell shares one value. Cached and Store connect the map to a
// durable cache the consumer keeps.
//
// The resolver logs through go-log: coordinates, cache hits, requests, and
// their outcome at debug, and fallbacks at warn. It logs nothing else about
// the caller. WithLogger replaces the logger.
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

	astronomy "github.com/Bugs5382/go-astronomy"
)

const (
	// DefaultBaseURL is the Open-Meteo elevation endpoint.
	DefaultBaseURL = "https://api.open-meteo.com/v1/elevation"
	// DefaultRound is the default cache grid, in degrees.
	DefaultRound = 0.01
	// DefaultTimeout bounds each lookup, on top of any deadline on the
	// caller's context.
	DefaultTimeout = 10 * time.Second
)

// maxBody bounds how much of a reply is read; one elevation is a few dozen
// bytes.
const maxBody = 64 << 10

// Resolver is an astronomy.ElevationResolver backed by the Open-Meteo
// elevation API. Build it with New; it is safe for concurrent use. Two
// concurrent misses on one cell may both reach the service and store the same
// value.
type Resolver struct {
	http    *http.Client
	base    string
	round   float64
	timeout time.Duration
	log     log.Logger

	mu    sync.RWMutex
	cache map[cell]astronomy.Height
}

var _ astronomy.ElevationResolver = (*Resolver)(nil)

// cell is a cache key: a coordinate in whole grid steps.
type cell struct{ lat, lng int64 }

// Option configures a Resolver.
type Option func(*Resolver)

// WithHTTPClient sets the HTTP client for requests. The default is a plain
// client; the per-lookup timeout applies either way.
func WithHTTPClient(c *http.Client) Option {
	return func(r *Resolver) {
		if c != nil {
			r.http = c
		}
	}
}

// WithTimeout sets how long one lookup may take before it falls back to sea
// level. A non-positive value keeps DefaultTimeout.
func WithTimeout(d time.Duration) Option {
	return func(r *Resolver) {
		if d > 0 {
			r.timeout = d
		}
	}
}

// WithBaseURL sets the endpoint, for a self-hosted Open-Meteo or a test
// server. An empty value keeps DefaultBaseURL.
func WithBaseURL(u string) Option {
	return func(r *Resolver) {
		if u != "" {
			r.base = u
		}
	}
}

// WithRound sets the cache grid in degrees. A non-positive or non-finite value
// keeps DefaultRound.
func WithRound(deg float64) Option {
	return func(r *Resolver) {
		if deg > 0 && !math.IsInf(deg, 0) {
			r.round = deg
		}
	}
}

// WithLogger replaces the go-log logger the resolver logs through.
func WithLogger(l log.Logger) Option {
	return func(r *Resolver) {
		if l != nil {
			r.log = l
		}
	}
}

// New returns a Resolver with the options applied.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		http:    &http.Client{},
		base:    DefaultBaseURL,
		round:   DefaultRound,
		timeout: DefaultTimeout,
		cache:   map[cell]astronomy.Height{},
	}
	for _, o := range opts {
		o(r)
	}
	if r.log == nil {
		r.log = log.NewLogger("go-astronomy")
	}
	return r
}

var defaultResolver = sync.OnceValue(func() *Resolver { return New() })

// Default returns the shared Resolver with the default options, built on
// first use, so its cache lasts for the life of the process.
func Default() *Resolver { return defaultResolver() }

// ResolveObserver returns an observer at the coordinate with its height from
// the Open-Meteo API (through Default), and the source: SourceOpenMeteo, or
// SourceSeaLevel when the lookup failed and the height fell back to sea level.
// The time zone is left nil (UTC); set TZ on the result. The error is only for
// a cancelled or expired context.
func ResolveObserver(ctx context.Context, lat, lng float64) (astronomy.Observer, string, error) {
	return astronomy.ResolveObserverWith(ctx, Default(), lat, lng)
}

// key returns the cache cell of a coordinate and the coordinate of the cell.
func (r *Resolver) key(lat, lng float64) (cell, float64, float64) {
	k := cell{int64(math.Round(lat / r.round)), int64(math.Round(lng / r.round))}
	return k, float64(k.lat) * r.round, float64(k.lng) * r.round
}

// Elevation returns the height of the ground at the coordinate from the cache
// or from Open-Meteo, with SourceOpenMeteo. Any failure answers SeaLevel with
// SourceSeaLevel and no error; the error is only for a done ctx.
func (r *Resolver) Elevation(ctx context.Context, lat, lng float64) (astronomy.Height, string, error) {
	if err := ctx.Err(); err != nil {
		return astronomy.SeaLevel, astronomy.SourceSeaLevel, err
	}
	if !(lat >= -90 && lat <= 90) || !(lng >= -180 && lng <= 180) {
		r.log.Warn("elevation lookup skipped: coordinate out of range; using sea level",
			log.F("lat", lat), log.F("lng", lng))
		return astronomy.SeaLevel, astronomy.SourceSeaLevel, nil
	}
	k, qlat, qlng := r.key(lat, lng)
	r.mu.RLock()
	h, ok := r.cache[k]
	r.mu.RUnlock()
	if ok {
		r.log.Debug("elevation cache hit", log.F("lat", qlat), log.F("lng", qlng), log.F("elevation_m", h.Meters()))
		return h, astronomy.SourceOpenMeteo, nil
	}
	r.log.Debug("elevation cache miss", log.F("lat", qlat), log.F("lng", qlng))

	h, err := r.fetch(ctx, qlat, qlng)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			r.log.Debug("elevation lookup stopped by the caller's context", log.F("lat", qlat), log.F("lng", qlng))
			return astronomy.SeaLevel, astronomy.SourceSeaLevel, ctxErr
		}
		r.log.Warn("elevation lookup failed; using sea level",
			log.F("lat", qlat), log.F("lng", qlng), log.F("reason", err.Error()))
		return astronomy.SeaLevel, astronomy.SourceSeaLevel, nil
	}
	r.Store(lat, lng, h)
	return h, astronomy.SourceOpenMeteo, nil
}

// Cached returns the cached height for the coordinate's cell and whether one
// is held, without any request.
func (r *Resolver) Cached(lat, lng float64) (astronomy.Height, bool) {
	k, _, _ := r.key(lat, lng)
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.cache[k]
	return h, ok
}

// Store records a height for the coordinate's cell, for seeding the cache from
// a durable store. A NaN or infinite height is ignored.
func (r *Resolver) Store(lat, lng float64, h astronomy.Height) {
	if h.Err() != nil {
		return
	}
	k, _, _ := r.key(lat, lng)
	r.mu.Lock()
	r.cache[k] = h
	r.mu.Unlock()
}

// reply is the Open-Meteo elevation response: one height per coordinate, or
// an error flag and reason.
type reply struct {
	Elevation []*float64 `json:"elevation"`
	Error     bool       `json:"error"`
	Reason    string     `json:"reason"`
}

var errNoValue = errors.New("no elevation in the reply")

// fetch asks Open-Meteo for one coordinate, bounded by the resolver timeout.
func (r *Resolver) fetch(ctx context.Context, lat, lng float64) (astronomy.Height, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	q := url.Values{}
	q.Set("latitude", coord(lat))
	q.Set("longitude", coord(lng))
	target := r.base + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return astronomy.SeaLevel, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-astronomy")

	start := time.Now()
	resp, err := r.http.Do(req) // #nosec G107 G704 -- the configured endpoint plus two numbers.
	if err != nil {
		r.log.Debug("elevation request failed", log.F("lat", lat), log.F("lng", lng),
			log.F("duration_ms", time.Since(start).Milliseconds()))
		return astronomy.SeaLevel, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	r.log.Debug("elevation request", log.F("lat", lat), log.F("lng", lng), log.F("status", resp.StatusCode),
		log.F("duration_ms", time.Since(start).Milliseconds()))
	if err != nil {
		return astronomy.SeaLevel, fmt.Errorf("read reply: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var rp reply
		_ = json.Unmarshal(body, &rp)
		return astronomy.SeaLevel, fmt.Errorf("status %d: %s", resp.StatusCode, rp.Reason)
	}
	var rp reply
	if err := json.Unmarshal(body, &rp); err != nil {
		return astronomy.SeaLevel, fmt.Errorf("decode reply: %w", err)
	}
	if rp.Error || len(rp.Elevation) != 1 || rp.Elevation[0] == nil {
		return astronomy.SeaLevel, errNoValue
	}
	h := astronomy.Meters(*rp.Elevation[0])
	if err := h.Err(); err != nil {
		return astronomy.SeaLevel, err
	}
	r.log.Debug("elevation fetched", log.F("lat", lat), log.F("lng", lng), log.F("elevation_m", h.Meters()))
	return h, nil
}

// coord formats a cell coordinate to the microdegree, dropping the float noise
// of multiplying the grid step back out (-104.99000000000001).
func coord(v float64) string {
	return strconv.FormatFloat(math.Round(v*1e6)/1e6, 'f', -1, 64)
}
