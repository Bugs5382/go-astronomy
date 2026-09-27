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
	"errors"
	"math"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestStaticElements checks the caller-supplied source answers by catalogue
// number and reports a missing one.
func TestStaticElements(t *testing.T) {
	t.Parallel()
	e := iss(t)
	src := satellite.StaticElements(e)
	got, err := src.Elements(context.Background(), 25544)
	if err != nil || got.SatNum != 25544 {
		t.Fatalf("Elements(25544) = %v, %v", got.SatNum, err)
	}
	if _, err := src.Elements(context.Background(), 20580); !errors.Is(err, satellite.ErrNoElements) {
		t.Errorf("Elements(20580) = %v, want ErrNoElements", err)
	}
	done, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := src.Elements(done, 25544); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}

// staleSource returns its elements with ErrStaleElements, as a fetcher does
// when it could not refresh a cached set.
type staleSource struct{ e satellite.Elements }

func (s staleSource) Elements(context.Context, int) (satellite.Elements, error) {
	return s.e, satellite.ErrStaleElements
}

// TestTracker checks a tracker asks its source for its own catalogue number
// and matches the engine, and passes a stale set on with its flag.
func TestTracker(t *testing.T) {
	t.Parallel()
	e := iss(t)
	tr := satellite.NewTracker(25544, "ISS (ZARYA)", satellite.ISSStandardMagnitude, satellite.StaticElements(e))
	if tr.CatalogNumber() != 25544 || tr.Name() != "ISS (ZARYA)" {
		t.Errorf("tracker %d %q", tr.CatalogNumber(), tr.Name())
	}
	obs := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	when := time.Date(2026, 9, 26, 4, 5, 3, 0, time.UTC)
	ctx := context.Background()
	got, err := tr.Position(ctx, obs, when)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := satellite.Position(obs, e, when)
	if got != want {
		t.Errorf("tracker %+v, engine %+v", got, want)
	}
	from := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	passes, err := tr.Passes(ctx, obs, from, from.Add(6*time.Hour))
	if err != nil || len(passes) == 0 {
		t.Fatalf("passes %v, %v", passes, err)
	}
	if p := passes[0]; p.Peak.Sunlit && math.IsNaN(p.Peak.Magnitude) {
		t.Error("the tracker's standard magnitude should give sunlit events a magnitude")
	}

	other := satellite.NewTracker(20580, "HST", 2.2, satellite.StaticElements(e))
	if _, err := other.Position(ctx, obs, when); !errors.Is(err, satellite.ErrNoElements) {
		t.Errorf("a source without the tracker's number: %v", err)
	}

	stale := satellite.NewTracker(25544, "ISS (ZARYA)", -1.8, staleSource{e})
	l, err := stale.Position(ctx, obs, when)
	if !errors.Is(err, satellite.ErrStaleElements) || l != want {
		t.Errorf("stale: %+v, %v", l, err)
	}
	ps, err := stale.Passes(ctx, obs, from, from.Add(6*time.Hour))
	if !errors.Is(err, satellite.ErrStaleElements) || len(ps) != len(passes) {
		t.Errorf("stale passes: %d, %v", len(ps), err)
	}
}
