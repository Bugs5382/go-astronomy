package moon

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// horizonRefraction is the atmospheric refraction at the horizon, in degrees
// (about 34 arc minutes). Combined with the Moon's apparent semidiameter it
// gives the geometric center altitude at which the upper limb touches the
// horizon, the threshold for rise and set.
const horizonRefraction = 0.5667

const (
	// riseSetSearchStep is the coarse interval used to bracket a horizon
	// crossing. The Moon's altitude changes slowly enough that ten minutes never
	// skips a crossing.
	riseSetSearchStep = 10 * time.Minute
	// riseSetSearchLimit bounds the forward search. It spans a full synodic month
	// so that a crossing is found even at high latitudes where the Moon can stay
	// above or below the horizon for many days before its declination swings back.
	riseSetSearchLimit = 30 * 24 * time.Hour
)

// NextRise returns the first instant strictly after t at which the Moon's upper
// limb rises above the horizon for the observer, and true. It returns the zero
// time and false when no rise occurs within the search window, which can happen
// at high latitudes. It returns a go-apperr coded error when the observer's
// latitude or longitude is out of range.
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error) {
	return nextCrossing(obs, t, true)
}

// NextSet returns the first instant strictly after t at which the Moon's upper
// limb sets below the horizon for the observer, and true. It returns the zero
// time and false when no set occurs within the search window, which can happen
// at high latitudes. It returns a go-apperr coded error when the observer's
// latitude or longitude is out of range.
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error) {
	return nextCrossing(obs, t, false)
}

// nextCrossing scans forward from t for the next horizon crossing in the given
// direction and refines it by bisection. The crossing function is the Moon's
// topocentric geometric center altitude minus the rise/set threshold, which is
// the negative of the horizon refraction plus the Moon's apparent semidiameter
// at that instant, lowered further by the horizon dip for the observer's
// height (issue 52). A rising crossing is a change from negative to
// non-negative; a setting crossing is the reverse.
func nextCrossing(obs astronomy.Observer, t time.Time, rising bool) (time.Time, bool, error) {
	if err := validateObserver(obs); err != nil {
		return time.Time{}, false, err
	}
	dip := earth.HorizonDip(obs.Height)
	f := func(when time.Time) float64 {
		hz, semiDeg := topocentric(obs, when)
		return hz.Altitude - (-(horizonRefraction + semiDeg + dip))
	}

	prev := f(t)
	deadline := t.Add(riseSetSearchLimit)
	for when := t.Add(riseSetSearchStep); !when.After(deadline); when = when.Add(riseSetSearchStep) {
		cur := f(when)
		up := prev < 0 && cur >= 0
		down := prev >= 0 && cur < 0
		if (rising && up) || (!rising && down) {
			return bisectCrossing(f, when.Add(-riseSetSearchStep), when), true, nil
		}
		prev = cur
	}
	return time.Time{}, false, nil
}

// bisectCrossing refines the horizon crossing of f in [lo, hi], where f changes
// sign across the bracket, to sub-second precision.
func bisectCrossing(f func(time.Time) float64, lo, hi time.Time) time.Time {
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
