package horizons_test

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
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
	"github.com/Bugs5382/go-astronomy/satellite/horizons"
)

// fixtures maps a request (target, start, stop) to the recorded reply.
var fixtures = map[string]string{
	"-170|2026-09-03 20:00|2026-10-04 04:00": "jwst-window.txt",
	"-211|2026-09-03 20:00|2026-10-04 04:00": "roman-window-full.txt",
	"-211|2026-10-03 20:00|2026-11-03 04:00": "roman-beyond-coverage.txt",
	"-211|2026-10-03 20:00|2026-10-19 12:00": "roman-clipped.txt",
}

// fake plays the Horizons API from the recorded fixtures and counts requests.
type fake struct {
	srv      *httptest.Server
	calls    atomic.Int64
	mu       sync.Mutex
	override func(w http.ResponseWriter, r *http.Request) bool
	queries  []string
}

func unquote(s string) string { return strings.Trim(s, "'") }

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		q := r.URL.Query()
		k := unquote(q.Get("COMMAND")) + "|" + unquote(q.Get("START_TIME")) + "|" + unquote(q.Get("STOP_TIME"))
		f.mu.Lock()
		f.queries = append(f.queries, k)
		override := f.override
		f.mu.Unlock()
		if override != nil && override(w, r) {
			return
		}
		if q.Get("CENTER") != "'500@399'" || q.Get("QUANTITIES") != "'2,20'" || q.Get("STEP_SIZE") != "'60 m'" {
			http.Error(w, "unexpected query "+r.URL.RawQuery, http.StatusBadRequest)
			return
		}
		name, ok := fixtures[k]
		if !ok {
			http.Error(w, "no fixture for "+k, http.StatusBadRequest)
			return
		}
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func newClient(f *fake, opts ...horizons.Option) *horizons.Client {
	return horizons.New(append([]horizons.Option{horizons.WithBaseURL(f.srv.URL)}, opts...)...)
}

// row is one line of a recorded Horizons table: RA, Dec, then any further
// numeric columns.
type row struct {
	when time.Time
	vals []float64
}

func readTable(t *testing.T, name string) []row {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []row
	in := false
	sc := bufio.NewScanner(f)
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
		when, err := time.Parse("2006-Jan-02 15:04", strings.TrimSpace(p[0]))
		if err != nil {
			t.Fatal(err)
		}
		var v []float64
		for _, s := range p[3:] {
			if s = strings.TrimSpace(s); s != "" {
				x, err := strconv.ParseFloat(s, 64)
				if err != nil {
					t.Fatal(err)
				}
				v = append(v, x)
			}
		}
		out = append(out, row{when, v})
	}
	return out
}

func sepArcsec(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(lat1*r)*math.Sin(lat2*r) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Acos(math.Min(1, c)) / r * 3600
}

// TestInterpolationAccuracy interpolates the hourly window at every point of
// the ten-minute dense fixture and compares: the error of the eight-point
// Lagrange interpolation over the window.
func TestInterpolationAccuracy(t *testing.T) {
	t.Parallel()
	c := newClient(newFake(t))
	var worstPlace, worstKm float64
	for _, d := range readTable(t, "jwst-dense.txt") {
		g, err := c.Geocentric(context.Background(), "-170", d.when)
		if err != nil {
			t.Fatal(err)
		}
		worstPlace = math.Max(worstPlace, sepArcsec(g.RA, g.Dec, d.vals[0], d.vals[1]))
		worstKm = math.Max(worstKm, math.Abs(g.DistanceKm-d.vals[2]*149597870.7))
	}
	t.Logf("interpolation error over 2026-10-01 to 10-03 at 10 min: %.2e arcsec, %.2e km", worstPlace, worstKm)
	if worstPlace > interpolationTolArcsec || worstKm > 1 {
		t.Errorf("interpolation error %.3g arcsec, %.3g km", worstPlace, worstKm)
	}
}

// TestPositionAgainstHorizonsTopocentric checks the local topocentric
// correction and horizontal conversion against Horizons' own topocentric
// places for Greenwich.
func TestPositionAgainstHorizonsTopocentric(t *testing.T) {
	t.Parallel()
	c := newClient(newFake(t))
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005}
	var worstEq, worstHz, worstKm float64
	for _, h := range readTable(t, "jwst-greenwich.txt") {
		p, err := c.Position(context.Background(), "-170", greenwich, h.when)
		if err != nil {
			t.Fatal(err)
		}
		worstEq = math.Max(worstEq, sepArcsec(p.RA, p.Dec, h.vals[0], h.vals[1]))
		worstHz = math.Max(worstHz, sepArcsec(-p.Azimuth, p.Altitude, -h.vals[2], h.vals[3]))
		worstKm = math.Max(worstKm, math.Abs(p.RangeKm-h.vals[4]*149597870.7))
	}
	t.Logf("topocentric: RA/Dec %.2f arcsec, alt/az %.2f arcsec, range %.1f km", worstEq, worstHz, worstKm)
	if worstEq > 2 || worstHz > 5 || worstKm > 10 {
		t.Errorf("topocentric off Horizons: RA/Dec %.2f arcsec, alt/az %.2f arcsec, range %.1f km", worstEq, worstHz, worstKm)
	}
}

// TestCacheHit checks the window is fetched once for every instant inside it.
func TestCacheHit(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	c := newClient(f)
	ctx := context.Background()
	for _, at := range []time.Time{
		time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 25, 13, 17, 0, 0, time.UTC),
		time.Date(2026, 10, 3, 23, 0, 0, 0, time.UTC),
	} {
		if _, err := c.Geocentric(ctx, "-170", at); err != nil {
			t.Fatal(err)
		}
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

// TestWindowRefetchAndCoverage checks an instant outside the cached window
// fetches the next window, and that a window running past the trajectory's
// end is clipped to it and retried once.
func TestWindowRefetchAndCoverage(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	c := newClient(f)
	ctx := context.Background()
	if _, err := c.Geocentric(ctx, "-211", time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	g, err := c.Geocentric(ctx, "-211", time.Date(2026, 10, 10, 6, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next window: %v (requests %v)", err, f.queries)
	}
	if f.calls.Load() != 3 {
		t.Errorf("%d requests %v, want the first window, the refused next one, and its clipped retry", f.calls.Load(), f.queries)
	}
	if g.DistanceKm < 1e6 || g.DistanceKm > 2e6 {
		t.Errorf("Roman at %.0f km", g.DistanceKm)
	}
	// Past the end of the trajectory there is nothing to interpolate.
	_, err = c.Geocentric(ctx, "-211", time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, horizons.ErrOutsideCoverage) {
		t.Errorf("after the trajectory ends: %v", err)
	}
}

// TestFailures checks each failure is an ErrFetch error.
func TestFailures(t *testing.T) {
	t.Parallel()
	cases := map[string]func(w http.ResponseWriter, r *http.Request) bool{
		"timeout": func(w http.ResponseWriter, r *http.Request) bool {
			select {
			case <-time.After(2 * time.Second):
			case <-r.Context().Done():
			}
			return true
		},
		"non-200": func(w http.ResponseWriter, r *http.Request) bool {
			http.Error(w, "server error", http.StatusInternalServerError)
			return true
		},
		"no table": func(w http.ResponseWriter, r *http.Request) bool {
			_, _ = w.Write([]byte("API VERSION: 1.2\nNo matches found.\n"))
			return true
		},
		"bad row": func(w http.ResponseWriter, r *http.Request) bool {
			_, _ = w.Write([]byte("$$SOE\n 2026-Sep-03 20:00, , , x, y, z, w,\n$$EOE\n"))
			return true
		},
	}
	for name, fn := range cases {
		f := newFake(t)
		f.override = fn
		c := newClient(f, horizons.WithTimeout(100*time.Millisecond))
		_, err := c.Geocentric(context.Background(), "-170", time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
		if !errors.Is(err, horizons.ErrFetch) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestSharedCache checks two clients on one pluggable cache share a window.
func TestSharedCache(t *testing.T) {
	t.Parallel()
	f := newFake(t)
	shared := satellite.NewMemoryCache()
	a := newClient(f, horizons.WithCache(shared))
	b := newClient(f, horizons.WithCache(shared))
	at := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	ga, _ := a.Geocentric(context.Background(), "-170", at)
	gb, err := b.Geocentric(context.Background(), "-170", at)
	if err != nil || ga != gb || f.calls.Load() != 1 {
		t.Errorf("second client: %v, %d requests", err, f.calls.Load())
	}
}

// TestContext checks a cancelled context stops the fetch with its error.
func TestContext(t *testing.T) {
	t.Parallel()
	c := newClient(newFake(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Geocentric(ctx, "-170", time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}

// TestInvalidObserver checks the coded observer errors.
func TestInvalidObserver(t *testing.T) {
	t.Parallel()
	c := newClient(newFake(t))
	if _, err := c.Position(context.Background(), "-170", astronomy.Observer{Lat: 91}, time.Now()); !errors.Is(err, horizons.ErrInvalidLatitude) {
		t.Errorf("latitude 91: %v", err)
	}
}
