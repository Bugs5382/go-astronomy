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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/openmeteo"
)

// fake is a stand-in for the Open-Meteo elevation API. It answers with the
// given status and body after an optional delay, and records every query.
type fake struct {
	srv   *httptest.Server
	calls atomic.Int64
	last  atomic.Value
}

func newFake(t *testing.T, status int, body string, delay time.Duration) *fake {
	t.Helper()
	f := &fake{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.last.Store(r.URL.Query())
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func TestSuccessAndCacheHit(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[1597.0]}`, 0)
	r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL))
	ctx := context.Background()

	h, src, err := r.Elevation(ctx, 39.7392, -104.9903)
	if err != nil || h != astronomy.Meters(1597) || src != astronomy.SourceOpenMeteo {
		t.Fatalf("Elevation = %v %q %v", h, src, err)
	}
	q := f.last.Load().(url.Values)
	if q.Get("latitude") != "39.74" || q.Get("longitude") != "-104.99" {
		t.Errorf("queried %v, want the rounded cell 39.74, -104.99", q)
	}
	// Another point in the same 0.01 degree cell is a cache hit.
	h, src, err = r.Elevation(ctx, 39.7401, -104.9897)
	if err != nil || h != astronomy.Meters(1597) || src != astronomy.SourceOpenMeteo {
		t.Fatalf("second Elevation = %v %q %v", h, src, err)
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
	if got, ok := r.Cached(39.74, -104.99); !ok || got != astronomy.Meters(1597) {
		t.Errorf("Cached = %v %v", got, ok)
	}
}

// TestFailuresFallBackToSeaLevel checks every kind of lookup failure answers
// sea level with SourceSeaLevel and no error, and is not cached.
func TestFailuresFallBackToSeaLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		body   string
		delay  time.Duration
	}{
		{"timeout", http.StatusOK, `{"elevation":[1597.0]}`, 2 * time.Second},
		{"bad json", http.StatusOK, `<html>`, 0},
		{"non-200", http.StatusInternalServerError, `{"error":true,"reason":"down"}`, 0},
		{"400 reason", http.StatusBadRequest, `{"error":true,"reason":"Latitude must be in range of -90 to 90°. Given: 91.0."}`, 0},
		{"no value", http.StatusOK, `{"elevation":[]}`, 0},
		{"null value", http.StatusOK, `{"elevation":[null]}`, 0},
	}
	for _, tc := range cases {
		f := newFake(t, tc.status, tc.body, tc.delay)
		r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL), openmeteo.WithTimeout(100*time.Millisecond))
		start := time.Now()
		h, src, err := r.Elevation(context.Background(), 10, 20)
		if err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
			t.Errorf("%s: Elevation = %v %q %v, want sea level with no error", tc.name, h, src, err)
		}
		if tc.name == "timeout" && time.Since(start) > time.Second {
			t.Errorf("timeout: took %v, want about the 100 ms timeout", time.Since(start))
		}
		if _, ok := r.Cached(10, 20); ok {
			t.Errorf("%s: a failure was cached", tc.name)
		}
	}
}

// TestInvalidCoordinateFallsBack checks an out-of-range coordinate answers
// sea level without a request.
func TestInvalidCoordinateFallsBack(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[1]}`, 0)
	r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL))
	for _, c := range [][2]float64{{91, 0}, {0, 181}} {
		h, src, err := r.Elevation(context.Background(), c[0], c[1])
		if err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
			t.Errorf("%v: %v %q %v", c, h, src, err)
		}
	}
	if f.calls.Load() != 0 {
		t.Error("an invalid coordinate reached the service")
	}
}

// TestContextErrors checks the error is returned only for a cancelled or
// expired caller context.
func TestContextErrors(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[1]}`, 2*time.Second)
	r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL))

	done, cancel := context.WithCancel(context.Background())
	cancel()
	if h, src, err := r.Elevation(done, 1, 1); !errors.Is(err, context.Canceled) || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("cancelled: %v %q %v", h, src, err)
	}
	expiring, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	if _, _, err := r.Elevation(expiring, 1, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("caller deadline: %v, want context.DeadlineExceeded", err)
	}
}

// TestInjectedHTTPClient checks requests go through the client given.
func TestInjectedHTTPClient(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[42.5]}`, 0)
	var used atomic.Int64
	client := &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		used.Add(1)
		return http.DefaultTransport.RoundTrip(req)
	})}
	r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL), openmeteo.WithHTTPClient(client))
	if h, _, _ := r.Elevation(context.Background(), 1, 1); h != astronomy.Meters(42.5) || used.Load() != 1 {
		t.Errorf("height %v, client used %d times", h, used.Load())
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestChainWithOpenMeteo is the api-header shape: a caller value first, the
// lookup second, sea level last.
func TestChainWithOpenMeteo(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[10.0]}`, 0)
	om := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL))
	ctx := context.Background()

	chain := astronomy.ChainElevation(astronomy.CallerElevation(astronomy.Meters(21), true), om)
	if h, src, _ := chain.Elevation(ctx, 40.71, -74.01); h != astronomy.Meters(21) || src != astronomy.SourceCaller {
		t.Errorf("with a caller value: %v %q", h, src)
	}
	if f.calls.Load() != 0 {
		t.Error("the lookup ran although the caller had a value")
	}
	chain = astronomy.ChainElevation(astronomy.CallerElevation(astronomy.SeaLevel, false), om)
	if h, src, _ := chain.Elevation(ctx, 40.71, -74.01); h != astronomy.Meters(10) || src != astronomy.SourceOpenMeteo {
		t.Errorf("without a caller value: %v %q", h, src)
	}
	broken := openmeteo.New(openmeteo.WithBaseURL(newFake(t, http.StatusBadGateway, `{}`, 0).srv.URL))
	chain = astronomy.ChainElevation(astronomy.CallerElevation(astronomy.SeaLevel, false), broken)
	if h, src, err := chain.Elevation(ctx, 40.71, -74.01); err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("everything failed: %v %q %v", h, src, err)
	}
}

// TestResolveObserverWithOpenMeteo checks the helper against a fake API; the
// package-level ResolveObserver uses the real service and is not run here.
func TestResolveObserverWithOpenMeteo(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[1597.0]}`, 0)
	obs, src, err := astronomy.ResolveObserverWith(context.Background(), openmeteo.New(openmeteo.WithBaseURL(f.srv.URL)), 39.74, -104.99)
	if err != nil || src != astronomy.SourceOpenMeteo || obs.Height != astronomy.Meters(1597) || obs.Lat != 39.74 {
		t.Errorf("observer %+v %q %v", obs, src, err)
	}
}

// TestConcurrentNew builds resolvers from many goroutines at once, each with
// its own default logger. Under the race detector this needs go-log v1.2.1 or
// later, where the base logger is stored atomically (Bugs5382/go-log issue 5).
func TestConcurrentNew(t *testing.T) {
	t.Parallel()
	f := newFake(t, http.StatusOK, `{"elevation":[1.0]}`, 0)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := openmeteo.New(openmeteo.WithBaseURL(f.srv.URL))
			if _, _, err := r.Elevation(context.Background(), 1, 1); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
