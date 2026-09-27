package satellite

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

// A tracker: one satellite, identified by its catalogue number, with its
// element sets from an explicit source.

import (
	"context"
	"errors"
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// Tracker follows one satellite. The named packages (satellite/iss,
// satellite/hubble, satellite/tiangong) build one with their catalogue number;
// NewTracker builds one for any other satellite.
type Tracker struct {
	catalog int
	name    string
	stdMag  float64
	src     ElementSource
}

// NewTracker returns a tracker for the satellite with the given NORAD
// catalogue number and name, taking its element sets from src. stdMag is its
// standard magnitude (see Look.Magnitude), or NaN when none is known.
func NewTracker(catalog int, name string, stdMag float64, src ElementSource) *Tracker {
	return &Tracker{catalog: catalog, name: name, stdMag: stdMag, src: src}
}

// CatalogNumber returns the satellite's NORAD catalogue number.
func (t *Tracker) CatalogNumber() int { return t.catalog }

// Name returns the satellite's name.
func (t *Tracker) Name() string { return t.name }

// StandardMagnitude returns the satellite's standard magnitude, or NaN.
func (t *Tracker) StandardMagnitude() float64 { return t.stdMag }

// Elements returns the current element set from the source.
func (t *Tracker) Elements(ctx context.Context) (Elements, error) {
	return t.src.Elements(ctx, t.catalog)
}

// Position returns where the satellite is in the observer's sky at t, from the
// source's current element set. When the source could only give a stale set,
// the Look is filled in from it and the error wraps ErrStaleElements.
func (t *Tracker) Position(ctx context.Context, obs astronomy.Observer, at time.Time) (Look, error) {
	e, err := t.Elements(ctx)
	stale := errors.Is(err, ErrStaleElements)
	if err != nil && !stale {
		return Look{}, err
	}
	l, perr := Position(obs, e, at)
	if perr != nil {
		return l, perr
	}
	return l, err
}

// Passes returns the satellite's passes over the observer between from and
// to, with DefaultPassOptions and the tracker's standard magnitude. A stale
// element set is used and reported as with Position.
func (t *Tracker) Passes(ctx context.Context, obs astronomy.Observer, from, to time.Time) ([]Pass, error) {
	opt := DefaultPassOptions()
	opt.StdMagnitude = t.stdMag
	return t.PassesWith(ctx, obs, from, to, opt)
}

// PassesWith is Passes with caller-supplied options.
func (t *Tracker) PassesWith(ctx context.Context, obs astronomy.Observer, from, to time.Time, opt PassOptions) ([]Pass, error) {
	e, err := t.Elements(ctx)
	stale := errors.Is(err, ErrStaleElements)
	if err != nil && !stale {
		return nil, err
	}
	p, perr := Passes(obs, e, from, to, opt)
	if perr != nil {
		return p, perr
	}
	return p, err
}

// Magnitude returns the apparent magnitude of a Look for this satellite, from
// its standard magnitude: NaN when none is known, +Inf in shadow.
func (t *Tracker) Magnitude(l Look) float64 {
	if math.IsNaN(t.stdMag) {
		return t.stdMag
	}
	return l.Magnitude(t.stdMag)
}
