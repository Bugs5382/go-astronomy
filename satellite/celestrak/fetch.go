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
	// Epoch is the set's epoch, and Age how old it was when this call
	// answered: how old the data behind a position or pass is.
	Epoch time.Time
	Age   time.Duration
}

// record is what the cache holds for one catalogue number.
type record struct {
	FetchedAt time.Time       `json:"fetched_at"`
	OMM       json.RawMessage `json:"omm"`
}

func key(catalog int) string { return "celestrak:gp:" + strconv.Itoa(catalog) }

// Elements returns the current set for the catalogue number, as
// satellite.ElementSource.
func (c *Client) Elements(ctx context.Context, catalog int) (satellite.Elements, error) {
	r, err := c.Fetch(ctx, catalog)
	return r.Elements, err
}

// Fetch returns the set for the catalogue number. With a set cached it
// answers at once, starting one background refresh when the set is older than
// the refresh age. With nothing cached it fetches, and callers arriving
// together share the one request; a failed first fetch is an error, and
// further calls get the same error without a request until MinRefetch has
// passed.
func (c *Client) Fetch(ctx context.Context, catalog int) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if cached, ok := c.load(ctx, catalog); ok {
		now := c.now()
		age := now.Sub(cached.FetchedAt)
		c.log.Debug("element set cache hit", log.F("catalog", catalog), log.F("age_s", int(age.Seconds())))
		if !c.never && age >= c.refreshAge {
			c.refreshInBackground(catalog)
		}
		return c.stamp(cached), nil
	}
	return c.fetchShared(ctx, catalog)
}

// stamp fills in the epoch and age of a result.
func (c *Client) stamp(r Result) Result {
	r.Epoch = r.Elements.Epoch()
	r.Age = c.now().Sub(r.Epoch)
	return r
}

// fetchShared makes the first request for a catalogue number, or waits on the
// one already under way.
func (c *Client) fetchShared(ctx context.Context, catalog int) (Result, error) {
	c.mu.Lock()
	if cl, ok := c.inFlight[catalog]; ok {
		c.mu.Unlock()
		select {
		case <-cl.done:
			return c.stamp(cl.res), cl.err
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}
	now := c.now()
	if last, ok := c.failures[catalog]; ok && now.Sub(last) < MinRefetch {
		c.mu.Unlock()
		return Result{}, fmt.Errorf("%w: the last request for %d failed at %s; retrying after %s", ErrFetch, catalog,
			last.Format(time.RFC3339), last.Add(MinRefetch).Format(time.RFC3339))
	}
	cl := &call{done: make(chan struct{})}
	c.inFlight[catalog] = cl
	c.attempts[catalog] = now
	c.mu.Unlock()

	cl.res, cl.err = c.fetchAndStore(ctx, catalog)
	c.mu.Lock()
	delete(c.inFlight, catalog)
	if cl.err == nil {
		delete(c.failures, catalog)
	} else if ctx.Err() == nil {
		c.failures[catalog] = now
	}
	c.mu.Unlock()
	close(cl.done)
	if cl.err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, ctxErr
		}
		c.log.Warn("element set fetch failed", log.F("catalog", catalog), log.F("reason", cl.err.Error()))
		return Result{}, cl.err
	}
	return c.stamp(cl.res), nil
}

// refreshInBackground starts one refresh for the catalogue number, unless one
// is under way or the last request was under MinRefetch ago. The caller
// already has its answer; a failure keeps the cached set.
func (c *Client) refreshInBackground(catalog int) {
	c.mu.Lock()
	now := c.now()
	_, busy := c.inFlight[catalog]
	last, tried := c.attempts[catalog]
	if busy || (tried && now.Sub(last) < MinRefetch) {
		c.mu.Unlock()
		return
	}
	cl := &call{done: make(chan struct{})}
	c.inFlight[catalog] = cl
	c.attempts[catalog] = now
	c.mu.Unlock()
	c.log.Debug("element set refresh started", log.F("catalog", catalog))
	go func() {
		// The refresh outlives the call that started it, so it has its own
		// context, bounded by the request timeout.
		ctx := context.Background()
		cl.res, cl.err = c.fetchAndStore(ctx, catalog)
		c.mu.Lock()
		delete(c.inFlight, catalog)
		c.mu.Unlock()
		close(cl.done)
		if cl.err != nil {
			c.log.Warn("element set refresh failed; keeping the cached set", log.F("catalog", catalog),
				log.F("retry_after", now.Add(MinRefetch).Format(time.RFC3339)), log.F("reason", cl.err.Error()))
			return
		}
		c.log.Debug("element set refreshed", log.F("catalog", catalog))
	}()
}

// fetchAndStore requests the set and caches it.
func (c *Client) fetchAndStore(ctx context.Context, catalog int) (Result, error) {
	r, raw, err := c.fetch(ctx, catalog)
	if err != nil {
		return Result{}, err
	}
	r.FetchedAt = c.now()
	c.store(ctx, catalog, record{FetchedAt: r.FetchedAt, OMM: raw})
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
		// No expiry: a set is replaced by the next refresh, never dropped.
		err = c.cache.Set(ctx, key(catalog), b, 0)
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
