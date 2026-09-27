package openmeteo_test

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
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/elevation"
	"github.com/Bugs5382/go-astronomy/elevation/openmeteo"
)

// server answers with the given status and body, shaped like the Open-Meteo
// elevation API's replies, and counts requests.
func server(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int64, *atomic.Value) {
	t.Helper()
	var n atomic.Int64
	var last atomic.Value
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		last.Store(r.URL.Query())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s, &n, &last
}

func TestElevationAndCache(t *testing.T) {
	t.Parallel()
	s, n, last := server(t, http.StatusOK, `{"elevation":[1609.0]}`)
	c := openmeteo.New(openmeteo.WithBaseURL(s.URL))
	ctx := context.Background()

	h, err := c.Elevation(ctx, 39.7392, -104.9903)
	if err != nil || h != 1609 {
		t.Fatalf("Elevation = %v, %v; want 1609", h, err)
	}
	q := last.Load().(url.Values)
	if q["latitude"][0] != "39.74" || q["longitude"][0] != "-104.99" {
		t.Errorf("queried %v, want the cell's rounded coordinate 39.74, -104.99", q)
	}
	// A point in the same 0.01 degree cell is served from the cache.
	if h, err := c.Elevation(ctx, 39.7401, -104.9897); err != nil || h != 1609 {
		t.Fatalf("second Elevation = %v, %v", h, err)
	}
	if got := n.Load(); got != 1 {
		t.Errorf("%d requests, want 1", got)
	}
	if h, ok := c.Cached(39.74, -104.99); !ok || h != 1609 {
		t.Errorf("Cached = %v, %v", h, ok)
	}
}

func TestRoundAndStore(t *testing.T) {
	t.Parallel()
	s, n, _ := server(t, http.StatusOK, `{"elevation":[3640.0]}`)
	c := openmeteo.New(openmeteo.WithBaseURL(s.URL), openmeteo.WithRound(0.05))
	c.Store(-16.50, -68.15, 3640)
	// -16.52 rounds to the same 0.05 degree cell as -16.50.
	if h, err := c.Elevation(context.Background(), -16.52, -68.16); err != nil || h != 3640 {
		t.Fatalf("Elevation = %v, %v", h, err)
	}
	if n.Load() != 0 {
		t.Errorf("a stored cell reached the service")
	}
}

func TestServiceErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"reason", http.StatusBadRequest, `{"error":true,"reason":"Latitude must be in range of -90 to 90°. Given: 91.0."}`},
		{"server", http.StatusInternalServerError, `{}`},
		{"garbage", http.StatusOK, `<html>`},
		{"empty", http.StatusOK, `{"elevation":[]}`},
		{"two", http.StatusOK, `{"elevation":[1,2]}`},
	}
	for _, tc := range cases {
		s, _, _ := server(t, tc.status, tc.body)
		c := openmeteo.New(openmeteo.WithBaseURL(s.URL))
		_, err := c.Elevation(context.Background(), 10, 10)
		code, _ := apperr.Code(err)
		if !errors.Is(err, elevation.ErrLookupFailed) || code != astronomy.CodeElevationLookup {
			t.Errorf("%s: error = %v (code %d), want ErrLookupFailed", tc.name, err, code)
		}
		if _, ok := c.Cached(10, 10); ok {
			t.Errorf("%s: a failure was cached", tc.name)
		}
	}
}

func TestInvalidCoordinate(t *testing.T) {
	t.Parallel()
	s, n, _ := server(t, http.StatusOK, `{"elevation":[0]}`)
	c := openmeteo.New(openmeteo.WithBaseURL(s.URL))
	if _, err := c.Elevation(context.Background(), 91, 0); !errors.Is(err, elevation.ErrInvalidLatitude) {
		t.Errorf("lat 91: %v", err)
	}
	if _, err := c.Elevation(context.Background(), 0, 181); !errors.Is(err, elevation.ErrInvalidLongitude) {
		t.Errorf("lng 181: %v", err)
	}
	if n.Load() != 0 {
		t.Errorf("an invalid coordinate reached the service")
	}
}

func TestCancelledContext(t *testing.T) {
	t.Parallel()
	s, _, _ := server(t, http.StatusOK, `{"elevation":[1]}`)
	c := openmeteo.New(openmeteo.WithBaseURL(s.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Elevation(ctx, 1, 1); !errors.Is(err, context.Canceled) || !errors.Is(err, elevation.ErrLookupFailed) {
		t.Errorf("error = %v, want context.Canceled inside ErrLookupFailed", err)
	}
}

func TestFill(t *testing.T) {
	t.Parallel()
	s, _, _ := server(t, http.StatusOK, `{"elevation":[1609.0]}`)
	c := openmeteo.New(openmeteo.WithBaseURL(s.URL))
	obs, err := elevation.Fill(context.Background(), c, astronomy.Observer{Lat: 39.74, Lng: -104.99})
	if err != nil || obs.Elevation != 1609 || obs.Lat != 39.74 {
		t.Fatalf("Fill = %+v, %v", obs, err)
	}
	s2, _, _ := server(t, http.StatusOK, `{"elevation":[`+strconv.Itoa(20000)+`]}`)
	c2 := openmeteo.New(openmeteo.WithBaseURL(s2.URL))
	if _, err := elevation.Fill(context.Background(), c2, astronomy.Observer{Lat: 1, Lng: 1}); !errors.Is(err, elevation.ErrLookupFailed) {
		t.Errorf("Fill with a 20 km height: %v, want ErrLookupFailed", err)
	}
}
