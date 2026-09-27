package planet

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
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// HorizonAltitude is the geometric altitude of a planet's centre, in degrees,
// when it sits on the horizon under standard refraction (34 arc minutes). The
// discs are arc seconds across, so the centre stands in for the planet.
const HorizonAltitude = -0.5667

const (
	// searchStep brackets crossings. No planet's altitude or hour angle turns
	// around within ten minutes except in grazing geometry near the poles.
	searchStep = 10 * time.Minute
	// searchLimit bounds the forward search: long enough for a planet that
	// stays up or down for days at high latitude to come back.
	searchLimit = 30 * 24 * time.Hour
)

// NextRise returns the first instant strictly after t at which the planet's
// centre rises above the horizon (HorizonAltitude), and true. It returns the
// zero time and false when no rise happens within 30 days, which can happen at
// high latitudes. It returns a coded error for an out-of-range observer or a
// Body that is not an observable planet.
func NextRise(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error) {
	return nextEvent(obs, b, t, func(r Result) float64 { return r.Altitude - HorizonAltitude }, true)
}

// NextSet returns the first instant strictly after t at which the planet's
// centre sets below the horizon, and true, as NextRise.
func NextSet(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error) {
	return nextEvent(obs, b, t, func(r Result) float64 { return r.Altitude - HorizonAltitude }, false)
}

// NextTransit returns the first instant strictly after t at which the planet
// crosses the observer's meridian to the south of the pole it circles, the
// upper culmination, and true. The transit is found whether or not the planet
// is above the horizon then.
func NextTransit(obs astronomy.Observer, b Body, t time.Time) (time.Time, bool, error) {
	f := func(r Result, when time.Time) float64 {
		// The local hour angle folded into (-180, 180]: it passes zero, from
		// negative to positive, at the upper culmination.
		h := angles.Normalize(julian.ApparentSiderealTime(when)+obs.Lng-r.RA+180) - 180
		return h
	}
	return scan(obs, b, t, f, true, true)
}

func nextEvent(obs astronomy.Observer, b Body, t time.Time, f func(Result) float64, rising bool) (time.Time, bool, error) {
	return scan(obs, b, t, func(r Result, _ time.Time) float64 { return f(r) }, rising, false)
}

// scan steps forward from t looking for f to change sign in the wanted
// direction and refines the crossing by bisection. For the hour angle, the
// wrap from +180 to -180 is a sign change too, so wrapped rejects any bracket
// where the function jumps by more than half a turn.
func scan(obs astronomy.Observer, b Body, t time.Time, f func(Result, time.Time) float64, rising, wrapped bool) (time.Time, bool, error) {
	if err := validateObserver(obs); err != nil {
		return time.Time{}, false, err
	}
	if err := validateTarget(b); err != nil {
		return time.Time{}, false, err
	}
	at := func(when time.Time) float64 { return f(position(obs, b, when), when) }

	prev := at(t)
	deadline := t.Add(searchLimit)
	for when := t.Add(searchStep); !when.After(deadline); when = when.Add(searchStep) {
		cur := at(when)
		up := prev < 0 && cur >= 0
		down := prev >= 0 && cur < 0
		if wrapped && math.Abs(cur-prev) > 180 {
			up, down = false, false
		}
		if (rising && up) || (!rising && down) {
			return bisect(at, when.Add(-searchStep), when), true, nil
		}
		prev = cur
	}
	return time.Time{}, false, nil
}

// bisect refines a sign change of f in [lo, hi] to under a second.
func bisect(f func(time.Time) float64, lo, hi time.Time) time.Time {
	fLo := f(lo)
	for i := 0; i < 60 && hi.Sub(lo) > time.Second; i++ {
		mid := lo.Add(hi.Sub(lo) / 2)
		fMid := f(mid)
		if (fLo < 0) == (fMid < 0) {
			lo, fLo = mid, fMid
		} else {
			hi = mid
		}
	}
	return lo.Add(hi.Sub(lo) / 2)
}
