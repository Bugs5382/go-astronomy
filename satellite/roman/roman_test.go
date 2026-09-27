package roman_test

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
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite/horizons"
	"github.com/Bugs5382/go-astronomy/satellite/roman"
)

// recorded maps a Horizons request (target, start, stop) to its reply.
var recorded = map[string]string{
	"-211|2026-09-03 20:00|2026-10-04 04:00": "roman-window-full.txt",
	"-211|2026-10-03 20:00|2026-11-03 04:00": "roman-beyond-coverage.txt",
	"-211|2026-10-03 20:00|2026-10-19 12:00": "roman-clipped.txt",
}

// server plays Horizons from the recorded replies and counts requests.
func server(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var n atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		q := r.URL.Query()
		k := strings.Trim(q.Get("COMMAND"), "'") + "|" + strings.Trim(q.Get("START_TIME"), "'") + "|" + strings.Trim(q.Get("STOP_TIME"), "'")
		name, ok := recorded[k]
		if !ok {
			http.Error(w, "no fixture for "+k, http.StatusBadRequest)
			return
		}
		b, _ := os.ReadFile("testdata/" + name)
		_, _ = w.Write(b)
	}))
	t.Cleanup(s.Close)
	return s, &n
}

// TestIdentity checks the package fixes its Horizons target.
func TestIdentity(t *testing.T) {
	t.Parallel()
	tr := roman.New(horizons.New())
	if roman.Target != "-211" || tr.Target() != roman.Target || tr.Name() != roman.Name {
		t.Errorf("%q %q", tr.Target(), tr.Name())
	}
}

// TestPosition checks a position from the recorded window: an L2 object is
// about 1.5 million km away, and every observer shares one request.
func TestPosition(t *testing.T) {
	t.Parallel()
	s, n := server(t)
	tr := roman.New(horizons.New(horizons.WithBaseURL(s.URL)))
	at := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	for _, obs := range []astronomy.Observer{{Lat: 51.48, Lng: 0}, {Lat: -33.87, Lng: 151.21}, {Lat: 39.74, Lng: -104.99}} {
		p, err := tr.Position(context.Background(), obs, at)
		if err != nil {
			t.Fatal(err)
		}
		if p.RangeKm < 1e6 || p.RangeKm > 2e6 {
			t.Errorf("%+v: range %.0f km", obs, p.RangeKm)
		}
	}
	if n.Load() != 1 {
		t.Errorf("%d requests for three observers, want 1", n.Load())
	}
}

// TestNoHorizons checks an unreachable Horizons is an error, not a guess.
func TestNoHorizons(t *testing.T) {
	t.Parallel()
	s, _ := server(t)
	s.Close()
	tr := roman.New(horizons.New(horizons.WithBaseURL(s.URL)))
	if _, err := tr.Position(context.Background(), astronomy.Observer{}, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)); !errors.Is(err, horizons.ErrFetch) {
		t.Errorf("unreachable: %v", err)
	}
}

// TestCoverageEnds checks an instant past Roman's published trajectory is
// reported as outside the coverage rather than extrapolated.
func TestCoverageEnds(t *testing.T) {
	t.Parallel()
	s, _ := server(t)
	tr := roman.New(horizons.New(horizons.WithBaseURL(s.URL)))
	if _, err := tr.Position(context.Background(), astronomy.Observer{}, time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Errorf("inside the clipped window: %v", err)
	}
	if _, err := tr.Position(context.Background(), astronomy.Observer{}, time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)); !errors.Is(err, horizons.ErrOutsideCoverage) {
		t.Errorf("after the trajectory: %v", err)
	}
}
