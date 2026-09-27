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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/Bugs5382/go-log"

	"github.com/Bugs5382/go-astronomy/satellite"
)

// ErrFetch is the cause when CelesTrak could not supply a set: the request
// failed or timed out, the reply was not 200, or it held no usable set.
var ErrFetch = errors.New("celestrak: fetch failed")

// maxBody bounds a reply; one element set is well under a kilobyte.
const maxBody = 1 << 20

// Result is one element set and where it came from.
type Result struct {
	Elements satellite.Elements
	// FetchedAt is when the set was fetched from CelesTrak.
	FetchedAt time.Time
	// Stale reports that the set is past its TTL and could not be refreshed.
	Stale bool
}

// record is what the cache holds for one catalogue number.
type record struct {
	FetchedAt time.Time       `json:"fetched_at"`
	OMM       json.RawMessage `json:"omm"`
}

func key(catalog int) string { return "celestrak:gp:" + strconv.Itoa(catalog) }

// Elements returns the current set for the catalogue number, as
// satellite.ElementSource. A stale set comes with an error wrapping
// satellite.ErrStaleElements.
func (c *Client) Elements(ctx context.Context, catalog int) (satellite.Elements, error) {
	r, err := c.Fetch(ctx, catalog)
	return r.Elements, err
}

// Fetch returns the set for the catalogue number from the cache while it is
// within its TTL, and from CelesTrak otherwise. When a refresh fails and a set
// was cached, that set is returned with Stale set and an error wrapping both
// satellite.ErrStaleElements and ErrFetch, and no new request is made for
// MinRefetch. With nothing cached, a failure returns only the error.
func (c *Client) Fetch(ctx context.Context, catalog int) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	now := c.now()
	cached, haveCache := c.load(ctx, catalog)
	if haveCache && now.Sub(cached.FetchedAt) < c.ttl {
		c.log.Debug("element set cache hit", log.F("catalog", catalog), log.F("age_s", int(now.Sub(cached.FetchedAt).Seconds())))
		return cached, nil
	}
	c.mu.Lock()
	lastFail, failedRecently := c.failed[catalog]
	failedRecently = failedRecently && now.Sub(lastFail) < MinRefetch
	c.mu.Unlock()
	if haveCache && failedRecently {
		cached.Stale = true
		c.log.Debug("element set refresh held back after a failure", log.F("catalog", catalog))
		return cached, fmt.Errorf("%w: %w: last refresh failed at %s", satellite.ErrStaleElements, ErrFetch, lastFail.Format(time.RFC3339))
	}

	r, raw, err := c.fetch(ctx, catalog)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, ctxErr
		}
		c.mu.Lock()
		c.failed[catalog] = now
		c.mu.Unlock()
		c.log.Warn("element set fetch failed", log.F("catalog", catalog), log.F("reason", err.Error()))
		if haveCache {
			cached.Stale = true
			return cached, fmt.Errorf("%w: %w", satellite.ErrStaleElements, err)
		}
		return Result{}, err
	}
	c.mu.Lock()
	delete(c.failed, catalog)
	c.mu.Unlock()
	r.FetchedAt = now
	c.store(ctx, catalog, record{FetchedAt: now, OMM: raw})
	return r, nil
}

// load returns the cached set, if the cache has one it can read.
func (c *Client) load(ctx context.Context, catalog int) (Result, bool) {
	b, ok, err := c.cache.Get(ctx, key(catalog))
	if err != nil {
		c.log.Warn("element cache read failed", log.F("catalog", catalog), log.F("reason", err.Error()))
		return Result{}, false
	}
	if !ok {
		return Result{}, false
	}
	var rec record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Result{}, false
	}
	e, err := parseOne(rec.OMM, catalog)
	if err != nil {
		return Result{}, false
	}
	return Result{Elements: e, FetchedAt: rec.FetchedAt}, true
}

func (c *Client) store(ctx context.Context, catalog int, rec record) {
	b, err := json.Marshal(rec)
	if err == nil {
		err = c.cache.Set(ctx, key(catalog), b, retention)
	}
	if err != nil {
		c.log.Warn("element cache write failed", log.F("catalog", catalog), log.F("reason", err.Error()))
	}
}

// fetch asks CelesTrak for one catalogue number.
func (c *Client) fetch(ctx context.Context, catalog int) (Result, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	q := url.Values{}
	q.Set("CATNR", strconv.Itoa(catalog))
	q.Set("FORMAT", "json")
	target := c.base + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Result{}, nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-astronomy")
	start := time.Now()
	resp, err := c.http.Do(req) // #nosec G107 G704 -- the configured endpoint plus a number.
	if err != nil {
		return Result{}, nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	c.log.Debug("element set request", log.F("catalog", catalog), log.F("status", resp.StatusCode),
		log.F("duration_ms", time.Since(start).Milliseconds()))
	if err != nil {
		return Result{}, nil, fmt.Errorf("%w: read reply: %w", ErrFetch, err)
	}
	if resp.StatusCode != http.StatusOK {
		return Result{}, nil, fmt.Errorf("%w: status %d: %s", ErrFetch, resp.StatusCode, bytes.TrimSpace(body))
	}
	e, err := parseOne(body, catalog)
	if err != nil {
		return Result{}, nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	return Result{Elements: e}, body, nil
}

// parseOne reads an OMM JSON reply and returns the set for the catalogue
// number.
func parseOne(raw []byte, catalog int) (satellite.Elements, error) {
	sets, err := satellite.ParseOMM(bytes.NewReader(raw))
	if err != nil {
		return satellite.Elements{}, err
	}
	for _, e := range sets {
		if e.SatNum == catalog {
			return e, nil
		}
	}
	return satellite.Elements{}, fmt.Errorf("reply has no set for catalogue number %d", catalog)
}
