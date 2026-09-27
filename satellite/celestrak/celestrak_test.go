package celestrak_test

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/satellite"
	"github.com/Bugs5382/go-astronomy/satellite/celestrak"
)

// fake plays CelesTrak's GP endpoint: it serves the recorded fixture for a
// catalogue number, or the configured failure, and counts requests.
type fake struct {
	srv    *httptest.Server
	calls  atomic.Int64
	mu     sync.Mutex
	status int
	body   []byte
	delay  time.Duration
}

func newFake(t *testing.T) *fake {
	t.Helper()
	f := &fake{status: http.StatusOK}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		f.mu.Lock()
		status, body, delay := f.status, f.body, f.delay
		f.mu.Unlock()
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		if body == nil {
			var err error
			body, err = os.ReadFile("testdata/gp-" + r.URL.Query().Get("CATNR") + ".json")
			if err != nil {
				http.Error(w, "No GP data found", http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fake) fail(status int, body string, delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.body, f.delay = status, []byte(body), delay
}

// clock is a settable time source.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) Add(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
func newClock() *clock               { return &clock{now: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)} }

func client(f *fake, c *clock, opts ...celestrak.Option) *celestrak.Client {
	base := []celestrak.Option{celestrak.WithBaseURL(f.srv.URL), celestrak.WithClock(c.Now)}
	return celestrak.New(append(base, opts...)...)
}

func TestSuccess(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	cl := client(f, c)
	r, err := cl.Fetch(context.Background(), 25544)
	if err != nil {
		t.Fatal(err)
	}
	if r.Elements.SatNum != 25544 || !r.FetchedAt.Equal(c.Now()) {
		t.Errorf("result %d fetched %s", r.Elements.SatNum, r.FetchedAt)
	}
	want := time.Date(2026, 9, 26, 20, 26, 13, 869024000, time.UTC)
	if !r.Elements.Epoch().Equal(want) || !r.Epoch.Equal(want) {
		t.Errorf("epoch %s and %s, want %s", r.Elements.Epoch(), r.Epoch, want)
	}
	if r.Age != c.Now().Sub(want) {
		t.Errorf("age %v, want %v", r.Age, c.Now().Sub(want))
	}
	e, err := cl.Elements(context.Background(), 25544)
	if err != nil || e.SatNum != 25544 {
		t.Errorf("Elements = %d, %v", e.SatNum, err)
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

// waitFor polls until cond holds or a second passes.
func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return true
		}
	}
	return cond()
}

// TestAnswersFromStaleCacheWithoutBlocking checks a set past the refresh age
// is answered at once, while one refresh runs in the background.
func TestAnswersFromStaleCacheWithoutBlocking(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	cl := client(f, c)
	ctx := context.Background()
	first, _ := cl.Fetch(ctx, 25544)
	c.Add(4 * 24 * time.Hour) // past the 3-day refresh age
	f.fail(http.StatusOK, "", 500*time.Millisecond)
	f.mu.Lock()
	f.body = nil
	f.mu.Unlock()
	start := time.Now()
	r, err := cl.Fetch(ctx, 25544)
	if err != nil || !r.FetchedAt.Equal(first.FetchedAt) {
		t.Fatalf("stale answer: %v, fetched %s", err, r.FetchedAt)
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("the stale answer took %v; it should not wait for the refresh", d)
	}
	if !waitFor(t, func() bool { return f.calls.Load() == 2 }) {
		t.Fatalf("%d requests, want the background refresh", f.calls.Load())
	}
	if !waitFor(t, func() bool { r, _ := cl.Fetch(ctx, 25544); return r.FetchedAt.Equal(c.Now()) }) {
		t.Error("the refreshed set never replaced the old one")
	}
}

// TestOneRefreshForConcurrentCallers checks many callers of a stale set start
// one refresh, and many callers of an empty cache share one fetch.
func TestOneRefreshForConcurrentCallers(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	f.fail(http.StatusOK, "", 200*time.Millisecond)
	f.mu.Lock()
	f.body = nil
	f.mu.Unlock()
	cl := client(f, c)
	ctx := context.Background()
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cl.Fetch(ctx, 25544); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n := f.calls.Load(); n != 1 {
		t.Errorf("empty cache, 20 callers: %d requests, want 1", n)
	}
	c.Add(4 * 24 * time.Hour)
	for range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = cl.Fetch(ctx, 25544) }()
	}
	wg.Wait()
	waitFor(t, func() bool { return f.calls.Load() >= 2 })
	time.Sleep(300 * time.Millisecond)
	if n := f.calls.Load(); n != 2 {
		t.Errorf("stale set, 20 callers: %d requests in all, want 2", n)
	}
}

// TestFailedRefreshKeepsTheOldSet checks a failed background refresh keeps
// the cached set and is not retried before the minimum interval.
func TestFailedRefreshKeepsTheOldSet(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	cl := client(f, c)
	ctx := context.Background()
	first, _ := cl.Fetch(ctx, 25544)
	c.Add(4 * 24 * time.Hour)
	f.fail(http.StatusServiceUnavailable, "busy", 0)
	if r, err := cl.Fetch(ctx, 25544); err != nil || !r.FetchedAt.Equal(first.FetchedAt) {
		t.Fatalf("stale answer: %v", err)
	}
	waitFor(t, func() bool { return f.calls.Load() == 2 })
	time.Sleep(50 * time.Millisecond)
	c.Add(time.Hour)
	if r, err := cl.Fetch(ctx, 25544); err != nil || !r.FetchedAt.Equal(first.FetchedAt) {
		t.Errorf("after the failure: %v, fetched %s, want the old set", err, r.FetchedAt)
	}
	time.Sleep(50 * time.Millisecond)
	if n := f.calls.Load(); n != 2 {
		t.Errorf("an hour after a failed refresh: %d requests, want no retry yet", n)
	}
	c.Add(90 * time.Minute)
	f.fail(http.StatusOK, "", 0)
	f.mu.Lock()
	f.body = nil
	f.mu.Unlock()
	_, _ = cl.Fetch(ctx, 25544)
	if !waitFor(t, func() bool { return f.calls.Load() == 3 }) {
		t.Errorf("2.5 h after the failure: %d requests, want the retry", f.calls.Load())
	}
}

// TestMinimumRefetch checks a refresh age below the minimum interval is
// raised to it.
func TestMinimumRefetch(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	cl := client(f, c, celestrak.WithRefreshAge(30*time.Minute))
	ctx := context.Background()
	_, _ = cl.Fetch(ctx, 25544)
	c.Add(90 * time.Minute)
	_, _ = cl.Fetch(ctx, 25544)
	time.Sleep(50 * time.Millisecond)
	if n := f.calls.Load(); n != 1 {
		t.Errorf("at 90 min with a 30 min refresh age: %d requests (the 2 h minimum applies)", n)
	}
	c.Add(time.Hour)
	_, _ = cl.Fetch(ctx, 25544)
	if !waitFor(t, func() bool { return f.calls.Load() == 2 }) {
		t.Errorf("at 2.5 h: %d requests, want the refresh", f.calls.Load())
	}
}

// TestNeverExpire checks the explicit mode fetches once and never again.
func TestNeverExpire(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	cl := client(f, c, celestrak.NeverExpire())
	ctx := context.Background()
	first, _ := cl.Fetch(ctx, 25544)
	for range 5 {
		c.Add(30 * 24 * time.Hour)
		r, err := cl.Fetch(ctx, 25544)
		if err != nil || !r.FetchedAt.Equal(first.FetchedAt) {
			t.Fatalf("%v, fetched %s", err, r.FetchedAt)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if n := f.calls.Load(); n != 1 {
		t.Errorf("%d requests over five months, want 1", n)
	}
}

// TestFailures checks a failed first fetch is an error, and is not retried
// before the minimum interval.
func TestFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		body   string
		delay  time.Duration
	}{
		{"timeout", http.StatusOK, `[]`, 2 * time.Second},
		{"non-200", http.StatusInternalServerError, "oops", 0},
		{"no data", http.StatusNotFound, "No GP data found", 0},
		{"bad data", http.StatusOK, `<html>`, 0},
		{"empty list", http.StatusOK, `[]`, 0},
	}
	for _, tc := range cases {
		f, c := newFake(t), newClock()
		f.fail(tc.status, tc.body, tc.delay)
		cl := client(f, c, celestrak.WithTimeout(100*time.Millisecond))
		start := time.Now()
		r, err := cl.Fetch(context.Background(), 25544)
		if !errors.Is(err, celestrak.ErrFetch) || r.Elements.SatNum != 0 {
			t.Errorf("%s: %+v, %v", tc.name, r, err)
		}
		if tc.name == "timeout" && time.Since(start) > time.Second {
			t.Errorf("timeout took %v", time.Since(start))
		}
		c.Add(time.Hour)
		if _, err := cl.Fetch(context.Background(), 25544); !errors.Is(err, celestrak.ErrFetch) || f.calls.Load() != 1 {
			t.Errorf("%s: an hour later: %v, %d requests, want the error without a new request", tc.name, err, f.calls.Load())
		}
	}
}

// TestWrongSatellite checks a reply for another catalogue number is refused.
func TestWrongSatellite(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	b, _ := os.ReadFile("testdata/gp-20580.json")
	f.fail(http.StatusOK, string(b), 0)
	if _, err := client(f, c).Fetch(context.Background(), 25544); !errors.Is(err, celestrak.ErrFetch) {
		t.Errorf("a Hubble reply for the ISS: %v", err)
	}
}

// TestSharedCache checks two clients on one pluggable cache share a fetch, as
// replicas sharing Redis would.
func TestSharedCache(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	shared := satellite.NewMemoryCache(satellite.WithCacheClock(c.Now))
	a := client(f, c, celestrak.WithCache(shared))
	b := client(f, c, celestrak.WithCache(shared))
	_, _ = a.Fetch(context.Background(), 20580)
	c.Add(time.Hour)
	r, err := b.Fetch(context.Background(), 20580)
	if err != nil || r.Elements.SatNum != 20580 || f.calls.Load() != 1 {
		t.Errorf("second client: %v, %d requests, want the cached set", err, f.calls.Load())
	}
	if !r.FetchedAt.Equal(c.Now().Add(-time.Hour)) {
		t.Errorf("fetched at %s, want the first client's fetch", r.FetchedAt)
	}
}

// TestContext checks a cancelled context stops the first fetch with its
// error.
func TestContext(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	f.fail(http.StatusOK, `[]`, 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client(f, c).Fetch(ctx, 25544); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}
