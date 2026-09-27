package astronomy_test

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

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// TestHeightConversion checks the two constructors against each other: one
// foot is exactly 0.3048 m, so 5000 ft is 1524 m.
func TestHeightConversion(t *testing.T) {
	t.Parallel()
	ft := astronomy.Feet(5000)
	if got := ft.Meters(); got != 1524 {
		t.Errorf("Feet(5000).Meters() = %v, want 1524", got)
	}
	m := astronomy.Meters(1524)
	if got := m.Feet(); math.Abs(got-5000) > 1e-9 {
		t.Errorf("Meters(1524).Feet() = %v, want 5000", got)
	}
	if ft != m {
		t.Errorf("Feet(5000) = %v and Meters(1524) = %v should be equal", ft, m)
	}
	if astronomy.SeaLevel.Meters() != 0 || (astronomy.Height{}) != astronomy.SeaLevel {
		t.Error("the zero Height is sea level")
	}
	if got := astronomy.Feet(36000).Meters(); math.Abs(got-10972.8) > 1e-9 {
		t.Errorf("Feet(36000) = %v m, want 10972.8", got)
	}
	if s := m.String(); s != "1524 m" {
		t.Errorf("String = %q", s)
	}
}

// TestHeightAcceptsAnyRealValue checks that below sea level, mountains, and
// cruising altitude are all valid as given.
func TestHeightAcceptsAnyRealValue(t *testing.T) {
	t.Parallel()
	for _, v := range []float64{-430, -10994, 0, 8849, 10972.8, 400000, -1e7} {
		if err := astronomy.Meters(v).Err(); err != nil {
			t.Errorf("Meters(%v): %v", v, err)
		}
		if err := astronomy.Feet(v).Err(); err != nil {
			t.Errorf("Feet(%v): %v", v, err)
		}
	}
}

// TestHeightRejectsNonFinite checks NaN and the infinities are reported with
// the coded ErrInvalidHeight.
func TestHeightRejectsNonFinite(t *testing.T) {
	t.Parallel()
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for name, h := range map[string]astronomy.Height{"Meters": astronomy.Meters(v), "Feet": astronomy.Feet(v)} {
			err := h.Err()
			code, _ := apperr.Code(err)
			if !errors.Is(err, astronomy.ErrInvalidHeight) || code != astronomy.CodeInvalidHeight {
				t.Errorf("%s(%v).Err() = %v (code %d)", name, v, err, code)
			}
		}
	}
}

// namedResolver answers with a fixed height and source, or fails the way a
// broken lookup does: sea level, reported as such.
type namedResolver struct {
	h      astronomy.Height
	source string
	calls  *int
}

func (n namedResolver) Elevation(ctx context.Context, lat, lng float64) (astronomy.Height, string, error) {
	if n.calls != nil {
		*n.calls++
	}
	if err := ctx.Err(); err != nil {
		return astronomy.SeaLevel, astronomy.SourceSeaLevel, err
	}
	return n.h, n.source, nil
}

// TestStaticAndCallerElevation checks the two resolvers that need no lookup.
func TestStaticAndCallerElevation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, src, err := astronomy.StaticElevation(astronomy.Meters(1609)).Elevation(ctx, 39.74, -104.99)
	if err != nil || h != astronomy.Meters(1609) || src != astronomy.SourceStatic {
		t.Errorf("static = %v %q %v", h, src, err)
	}
	h, src, err = astronomy.CallerElevation(astronomy.Meters(21), true).Elevation(ctx, 40.71, -74.01)
	if err != nil || h != astronomy.Meters(21) || src != astronomy.SourceCaller {
		t.Errorf("caller = %v %q %v", h, src, err)
	}
	h, src, err = astronomy.CallerElevation(astronomy.Meters(21), false).Elevation(ctx, 40.71, -74.01)
	if err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("caller without a value = %v %q %v", h, src, err)
	}
	// A non-finite height is no answer at all.
	h, src, _ = astronomy.StaticElevation(astronomy.Meters(math.NaN())).Elevation(ctx, 0, 0)
	if h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("static NaN = %v %q", h, src)
	}
}

// TestChainElevation checks the chain tries resolvers in order, reports the
// source that answered, and falls back to sea level only at the end.
func TestChainElevation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var laterCalls int
	failing := namedResolver{h: astronomy.SeaLevel, source: astronomy.SourceSeaLevel}
	later := namedResolver{h: astronomy.Meters(5), source: "later", calls: &laterCalls}

	chain := astronomy.ChainElevation(astronomy.CallerElevation(astronomy.Meters(21), true), later)
	if h, src, err := chain.Elevation(ctx, 40.71, -74.01); err != nil || h != astronomy.Meters(21) || src != astronomy.SourceCaller {
		t.Errorf("caller first = %v %q %v", h, src, err)
	}
	if laterCalls != 0 {
		t.Errorf("the chain asked a later resolver after an answer")
	}

	chain = astronomy.ChainElevation(astronomy.CallerElevation(astronomy.SeaLevel, false), failing, astronomy.StaticElevation(astronomy.Feet(5000)))
	if h, src, _ := chain.Elevation(ctx, 40.71, -74.01); h != astronomy.Feet(5000) || src != astronomy.SourceStatic {
		t.Errorf("fall through to static = %v %q", h, src)
	}

	chain = astronomy.ChainElevation(failing, astronomy.CallerElevation(astronomy.SeaLevel, false))
	if h, src, err := chain.Elevation(ctx, 40.71, -74.01); err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("everything failed = %v %q %v", h, src, err)
	}
	if h, src, err := astronomy.ChainElevation().Elevation(ctx, 0, 0); err != nil || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("empty chain = %v %q %v", h, src, err)
	}

	// The error is only for a cancelled or expired context.
	done, cancel := context.WithCancel(ctx)
	cancel()
	chain = astronomy.ChainElevation(later, astronomy.StaticElevation(astronomy.Meters(9)))
	if h, src, err := chain.Elevation(done, 0, 0); !errors.Is(err, context.Canceled) || h != astronomy.SeaLevel || src != astronomy.SourceSeaLevel {
		t.Errorf("cancelled = %v %q %v", h, src, err)
	}
	expired, cancel2 := context.WithTimeout(ctx, time.Nanosecond)
	defer cancel2()
	time.Sleep(time.Millisecond)
	if _, _, err := chain.Elevation(expired, 0, 0); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expired = %v", err)
	}
}

// TestResolveObserverWith checks the helper builds an observer from the
// coordinate and whatever the resolver found.
func TestResolveObserverWith(t *testing.T) {
	t.Parallel()
	obs, src, err := astronomy.ResolveObserverWith(context.Background(), astronomy.StaticElevation(astronomy.Feet(5000)), 40.71, -74.01)
	if err != nil || src != astronomy.SourceStatic {
		t.Fatalf("source %q, %v", src, err)
	}
	want := astronomy.Observer{Lat: 40.71, Lng: -74.01, Height: astronomy.Feet(5000)}
	if obs != want {
		t.Errorf("observer %+v, want %+v", obs, want)
	}
}
