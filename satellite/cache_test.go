package satellite_test

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
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestMemoryCache checks Get, Set, and expiry against an injected clock.
func TestMemoryCache(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	c := satellite.NewMemoryCache(satellite.WithCacheClock(func() time.Time { return now }))
	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "a"); ok || err != nil {
		t.Fatalf("empty Get = %v %v", ok, err)
	}
	if err := c.Set(ctx, "a", []byte("one"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if v, ok, _ := c.Get(ctx, "a"); !ok || string(v) != "one" {
		t.Fatalf("Get = %q %v", v, ok)
	}
	now = now.Add(59 * time.Minute)
	if _, ok, _ := c.Get(ctx, "a"); !ok {
		t.Error("expired before its expiry")
	}
	now = now.Add(2 * time.Minute)
	if _, ok, _ := c.Get(ctx, "a"); ok {
		t.Error("still there after its expiry")
	}
	// A zero expiry keeps the value until it is replaced.
	_ = c.Set(ctx, "b", []byte("kept"), 0)
	now = now.Add(1000 * time.Hour)
	if v, ok, _ := c.Get(ctx, "b"); !ok || string(v) != "kept" {
		t.Errorf("no-expiry value = %q %v", v, ok)
	}
	// Values are copies: changing what Get returned does not change the cache.
	v, _, _ := c.Get(ctx, "b")
	v[0] = 'X'
	if v2, _, _ := c.Get(ctx, "b"); string(v2) != "kept" {
		t.Errorf("cache shared its slice: %q", v2)
	}
}
