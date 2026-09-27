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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Bugs5382/go-log"
)

// sample is one table row: geocentric apparent right ascension and
// declination of date, in degrees, and the distance, in km.
type sample struct {
	RA, Dec, Km float64
}

// table is a fetched window: samples every Step from Start.
type table struct {
	Target  string        `json:"target"`
	Start   time.Time     `json:"start"`
	Step    time.Duration `json:"step"`
	Samples []sample      `json:"samples"`
}

// windowStart returns the start of the window that holds at: windows are
// aligned to whole multiples of the window length from the Unix epoch, so
// every instant in one maps to the same request and cache entry.
func (c *Client) windowStart(at time.Time) time.Time {
	w := c.window.Nanoseconds()
	n := at.UnixNano() / w
	if at.UnixNano() < 0 && at.UnixNano()%w != 0 {
		n--
	}
	return time.Unix(0, n*w).UTC()
}

func (c *Client) key(target string, start time.Time) string {
	return fmt.Sprintf("horizons:%s:%s:%d", target, start.Format(time.RFC3339), int64(c.step/time.Minute))
}

// tableFor returns the cached window that holds at, fetching it on a miss.
func (c *Client) tableFor(ctx context.Context, target string, at time.Time) (*table, error) {
	start := c.windowStart(at)
	k := c.key(target, start)
	if b, ok, err := c.cache.Get(ctx, k); err != nil {
		c.log.Warn("ephemeris cache read failed", log.F("target", target), log.F("reason", err.Error()))
	} else if ok {
		var t table
		if json.Unmarshal(b, &t) == nil && len(t.Samples) > 0 {
			c.log.Debug("ephemeris cache hit", log.F("target", target), log.F("window", start.Format(time.RFC3339)))
			return &t, nil
		}
	}
	from := start.Add(-pad * c.step)
	to := start.Add(c.window + pad*c.step)
	t, err := c.fetch(ctx, target, from, to)
	var cov *coverageError
	if errors.As(err, &cov) {
		// The trajectory ends (or begins) inside the window: clip to it,
		// one step inside the edge Horizons named, and ask once more.
		if !cov.after.IsZero() {
			to = cov.after.Add(-time.Nanosecond).Truncate(c.step)
		}
		if !cov.prior.IsZero() {
			from = cov.prior.Truncate(c.step).Add(c.step)
		}
		c.log.Debug("ephemeris window clipped to the trajectory", log.F("target", target),
			log.F("from", from.Format(time.RFC3339)), log.F("to", to.Format(time.RFC3339)))
		if !to.After(from) {
			return nil, fmt.Errorf("%w: %s", ErrOutsideCoverage, cov.msg)
		}
		t, err = c.fetch(ctx, target, from, to)
	}
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(t); err == nil {
		// Keep the window a while past its end; an instant inside it never
		// needs a new request.
		if err := c.cache.Set(ctx, k, b, c.window+48*time.Hour); err != nil {
			c.log.Warn("ephemeris cache write failed", log.F("target", target), log.F("reason", err.Error()))
		}
	}
	return t, nil
}

// coverageError reports Horizons refusing a span outside the trajectory.
type coverageError struct {
	msg          string
	prior, after time.Time
}

func (e *coverageError) Error() string { return e.msg }

// parseCoverage recognises Horizons' "No ephemeris for target ... after A.D.
// 2026-OCT-19 13:00:00.0000 UT" (or "prior to A.D.") reply.
func parseCoverage(body string) (*coverageError, bool) {
	i := strings.Index(body, "No ephemeris for target")
	if i < 0 {
		return nil, false
	}
	line := body[i:]
	if j := strings.IndexByte(line, '\n'); j >= 0 {
		line = line[:j]
	}
	e := &coverageError{msg: strings.TrimSpace(line)}
	for _, tag := range []string{" after A.D. ", " prior to A.D. "} {
		k := strings.Index(line, tag)
		if k < 0 {
			continue
		}
		f := strings.Fields(line[k+len(tag):])
		if len(f) < 2 {
			continue
		}
		when, err := time.Parse("2006-Jan-02 15:04:05", titleMonth(f[0])+" "+strings.SplitN(f[1], ".", 2)[0])
		if err != nil {
			continue
		}
		if tag == " after A.D. " {
			e.after = when
		} else {
			e.prior = when
		}
	}
	return e, true
}

// titleMonth turns "2026-OCT-19" into "2026-Oct-19" for time.Parse.
func titleMonth(s string) string {
	p := strings.Split(s, "-")
	if len(p) == 3 && len(p[1]) == 3 {
		p[1] = p[1][:1] + strings.ToLower(p[1][1:])
	}
	return strings.Join(p, "-")
}
