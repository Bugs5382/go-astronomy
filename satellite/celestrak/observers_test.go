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
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite/iss"
)

// TestOneFetchForManyObservers checks the cache holds element sets by
// catalogue number only: two observers far apart, asking 50 minutes apart for
// a position and for passes, are served from one fetch, and everything else
// is propagated locally.
func TestOneFetchForManyObservers(t *testing.T) {
	t.Parallel()
	f, c := newFake(t), newClock()
	tracker := iss.New(client(f, c))
	ctx := context.Background()

	denver := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	first := time.Date(2026, 9, 27, 3, 0, 0, 0, time.UTC)

	if _, err := tracker.Position(ctx, denver, first); err != nil {
		t.Fatal(err)
	}
	c.Add(50 * time.Minute)
	later := first.Add(50 * time.Minute)
	if _, err := tracker.Position(ctx, sydney, later); err != nil {
		t.Fatal(err)
	}
	if _, err := tracker.Passes(ctx, sydney, later, later.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	// A date days away is still propagated from the same set.
	if _, err := tracker.Position(ctx, denver, later.Add(72*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if n := f.calls.Load(); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}
