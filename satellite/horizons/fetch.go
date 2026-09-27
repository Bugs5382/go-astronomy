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
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Bugs5382/go-log"
)

// kmPerAU converts Horizons' distances.
const kmPerAU = 149597870.7

// maxBody bounds a reply; a 30-day hourly table is under 100 KB.
const maxBody = 16 << 20

// fetch requests the geocentric apparent place of target from from to to at
// the client's step, and parses the table.
func (c *Client) fetch(ctx context.Context, target string, from, to time.Time) (*table, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	q := url.Values{}
	for k, v := range map[string]string{
		"format": "text", "COMMAND": "'" + target + "'", "OBJ_DATA": "'NO'", "MAKE_EPHEM": "'YES'",
		"EPHEM_TYPE": "'OBSERVER'", "CENTER": "'500@399'", "QUANTITIES": "'2,20'", "ANG_FORMAT": "'DEG'",
		"EXTRA_PREC": "'YES'", "CSV_FORMAT": "'YES'", "TIME_TYPE": "'UT'",
		"START_TIME": "'" + from.UTC().Format("2006-01-02 15:04") + "'",
		"STOP_TIME":  "'" + to.UTC().Format("2006-01-02 15:04") + "'",
		"STEP_SIZE":  "'" + strconv.Itoa(int(c.step/time.Minute)) + " m'",
	} {
		q.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	req.Header.Set("User-Agent", "go-astronomy")
	start := time.Now()
	resp, err := c.http.Do(req) // #nosec G107 G704 -- the configured endpoint and fixed parameters.
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && ctxErr != context.DeadlineExceeded {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	c.log.Debug("ephemeris request", log.F("target", target), log.F("from", from.Format(time.RFC3339)),
		log.F("to", to.Format(time.RFC3339)), log.F("status", resp.StatusCode), log.F("duration_ms", time.Since(start).Milliseconds()))
	if err != nil {
		return nil, fmt.Errorf("%w: read reply: %w", ErrFetch, err)
	}
	body := string(b)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrFetch, resp.StatusCode, strings.TrimSpace(firstLine(body)))
	}
	if cov, ok := parseCoverage(body); ok {
		return nil, cov
	}
	t, err := parseTable(body, c.step)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	t.Target = target
	return t, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// parseTable reads the rows between $$SOE and $$EOE: date, two flag columns,
// RA and Dec in degrees, and the distance in au. The rows must run every step
// from the first.
func parseTable(body string, step time.Duration) (*table, error) {
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 64<<10), maxBody)
	in := false
	t := &table{Step: step}
	for sc.Scan() {
		l := sc.Text()
		switch {
		case strings.HasPrefix(l, "$$SOE"):
			in = true
			continue
		case strings.HasPrefix(l, "$$EOE"):
			in = false
		}
		if !in {
			continue
		}
		p := strings.Split(l, ",")
		if len(p) < 6 {
			return nil, fmt.Errorf("short row %q", l)
		}
		when, err := time.Parse("2006-Jan-02 15:04", strings.TrimSpace(p[0]))
		if err != nil {
			return nil, fmt.Errorf("row time %q: %w", p[0], err)
		}
		var v [3]float64
		for i, col := range []int{3, 4, 5} {
			if v[i], err = strconv.ParseFloat(strings.TrimSpace(p[col]), 64); err != nil || math.IsNaN(v[i]) {
				return nil, fmt.Errorf("row %q: column %d is not a number", l, col)
			}
		}
		if len(t.Samples) == 0 {
			t.Start = when.UTC()
		} else if !when.Equal(t.Start.Add(time.Duration(len(t.Samples)) * step)) {
			return nil, fmt.Errorf("row at %s breaks the %v step", when, step)
		}
		t.Samples = append(t.Samples, sample{RA: v[0], Dec: v[1], Km: v[2] * kmPerAU})
	}
	if len(t.Samples) < 8 {
		return nil, fmt.Errorf("reply has %d rows, want at least 8", len(t.Samples))
	}
	return t, nil
}
