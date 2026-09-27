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

import (
	"errors"
	"math"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// ErrInvalidPassWindow is the cause when a pass search's end is not after its
// start, or the window is longer than MaxPassWindow; the coded error carries
// astronomy.CodeInvalidPassWindow.
var ErrInvalidPassWindow = errors.New("satellite: pass window end not after start, or too long")

// MaxPassWindow is the longest span Passes searches in one call. Element sets
// for low orbits go stale within days, so a month is already generous.
const MaxPassWindow = 31 * 24 * time.Hour

// PassOptions configures Passes. The zero value is not the default; start
// from DefaultPassOptions.
type PassOptions struct {
	// MinAltitude is the altitude, in degrees, a pass starts and ends at: 0
	// for the geometric horizon, or higher to ignore low passes' ends.
	MinAltitude float64
	// DarkSunAltitude is the Sun's altitude, in degrees, below which the
	// observer counts as being in darkness: -6 (civil twilight) by default.
	DarkSunAltitude float64
	// Step is the scan interval. Default 10 s; a pass lasts minutes, so
	// longer steps can miss a short low one.
	Step time.Duration
	// StdMagnitude, when not NaN, is the satellite's standard magnitude (see
	// Look.Magnitude), used to fill in PassEvent.Magnitude. The default is
	// NaN: the element set does not carry it.
	StdMagnitude float64
}

// DefaultPassOptions returns the defaults: passes from the geometric horizon,
// darkness below -6 degrees of Sun altitude, a 10 s scan, and no magnitude.
func DefaultPassOptions() PassOptions {
	return PassOptions{MinAltitude: 0, DarkSunAltitude: -6, Step: 10 * time.Second, StdMagnitude: math.NaN()}
}

// PassEvent is the satellite at one moment of a pass.
type PassEvent struct {
	Time time.Time
	astronomy.Horizontal
	RangeKm float64
	Sunlit  bool
	// Magnitude is the apparent magnitude when PassOptions.StdMagnitude is
	// set and the satellite is sunlit, +Inf in shadow, and NaN otherwise.
	Magnitude float64
}

// Pass is one passage of the satellite above PassOptions.MinAltitude.
type Pass struct {
	// Rise, Peak, and Set are where the pass begins, culminates, and ends.
	Rise, Peak, Set PassEvent
	// Visible reports that for some part of the pass the satellite is sunlit
	// while the observer is in darkness. VisibleFrom and VisibleTo bound that
	// part (the first such span, which is almost always the only one).
	Visible                bool
	VisibleFrom, VisibleTo time.Time
	// ShadowEntry and ShadowExit are when the satellite enters or leaves the
	// Earth's shadow during the pass; the zero time when it does not.
	ShadowEntry, ShadowExit time.Time
}

// Passes returns every pass above opt.MinAltitude whose rise falls in [from,
// to), plus a pass already in progress at from, with its true rise before
// from. Crossings are found by a scan every opt.Step and refined by bisection
// to well under a second; the peak by ternary search. Errors are go-apperr
// coded: an out-of-range observer, an invalid window, or a propagation
// failure.
func Passes(obs astronomy.Observer, e Elements, from, to time.Time, opt PassOptions) ([]Pass, error) {
	if err := validateObserver(obs); err != nil {
		return nil, err
	}
	if !to.After(from) || to.Sub(from) > MaxPassWindow {
		return nil, apperr.Coded(astronomy.CodeInvalidPassWindow, ErrInvalidPassWindow)
	}
	if opt.Step <= 0 {
		opt.Step = 10 * time.Second
	}

	var firstErr error
	alt := func(t time.Time) float64 {
		l, err := look(obs, e, t)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return l.Altitude - opt.MinAltitude
	}

	var passes []Pass
	t := from
	prev := alt(t)
	if prev >= 0 {
		// Already up: walk back to the rise.
		rise := from
		for back := from.Add(-opt.Step); from.Sub(back) < 30*time.Minute; back = back.Add(-opt.Step) {
			if alt(back) < 0 {
				rise = bisect(alt, back, back.Add(opt.Step))
				break
			}
		}
		set, end := findSet(alt, from, opt.Step)
		passes = append(passes, buildPass(obs, e, rise, set, opt))
		t, prev = end, alt(end)
	}
	for t.Before(to) {
		next := t.Add(opt.Step)
		cur := alt(next)
		if prev < 0 && cur >= 0 {
			rise := bisect(alt, t, next)
			set, end := findSet(alt, next, opt.Step)
			passes = append(passes, buildPass(obs, e, rise, set, opt))
			t, prev = end, alt(end)
			continue
		}
		t, prev = next, cur
		if firstErr != nil {
			return passes, firstErr
		}
	}
	if firstErr != nil {
		return passes, firstErr
	}
	return passes, nil
}

// findSet scans forward from an instant above the threshold to the set, and
// returns the refined set and the scan instant just after it.
func findSet(alt func(time.Time) float64, from time.Time, step time.Duration) (time.Time, time.Time) {
	t := from
	// A pass never lasts a day; the cap only guards a geostationary object
	// that never sets.
	for limit := from.Add(24 * time.Hour); t.Before(limit); t = t.Add(step) {
		next := t.Add(step)
		if alt(next) < 0 {
			return bisect(alt, t, next), next
		}
	}
	return t, t
}

func buildPass(obs astronomy.Observer, e Elements, rise, set time.Time, opt PassOptions) Pass {
	elev := func(t time.Time) float64 {
		l, _ := look(obs, e, t)
		return l.Altitude
	}
	peak := ternaryMax(elev, rise, set)
	p := Pass{
		Rise: event(obs, e, rise, opt),
		Peak: event(obs, e, peak, opt),
		Set:  event(obs, e, set, opt),
	}

	// Walk the pass once a second for shadow crossings and the visible span,
	// refining each shadow crossing by bisection.
	lit := func(t time.Time) bool { l, _ := look(obs, e, t); return l.Sunlit }
	dark := func(t time.Time) bool { l, _ := look(obs, e, t); return l.SunAltitude < opt.DarkSunAltitude }
	const walk = time.Second
	prevLit := lit(rise)
	inFirst := false
	for t := rise.Add(walk); !t.After(set.Add(walk)); t = t.Add(walk) {
		if t.After(set) {
			t = set
		}
		l := lit(t)
		switch {
		case prevLit && !l && p.ShadowEntry.IsZero():
			p.ShadowEntry = bisectBool(lit, t.Add(-walk), t)
		case !prevLit && l && p.ShadowExit.IsZero():
			p.ShadowExit = bisectBool(lit, t.Add(-walk), t)
		}
		switch vis := l && dark(t); {
		case vis && !p.Visible:
			p.Visible, inFirst = true, true
			p.VisibleFrom = t.Add(-walk)
			if p.VisibleFrom.Before(rise) {
				p.VisibleFrom = rise
			}
			p.VisibleTo = t
		case vis && inFirst:
			p.VisibleTo = t
		case !vis:
			inFirst = false
		}
		prevLit = l
		if t.Equal(set) {
			break
		}
	}
	return p
}

func event(obs astronomy.Observer, e Elements, t time.Time, opt PassOptions) PassEvent {
	l, _ := look(obs, e, t)
	ev := PassEvent{Time: t, Horizontal: l.Horizontal, RangeKm: l.RangeKm, Sunlit: l.Sunlit, Magnitude: math.NaN()}
	if !math.IsNaN(opt.StdMagnitude) {
		ev.Magnitude = l.Magnitude(opt.StdMagnitude)
	}
	return ev
}

// bisect refines a sign change of f in [lo, hi] to under 10 ms.
func bisect(f func(time.Time) float64, lo, hi time.Time) time.Time {
	fLo := f(lo)
	for i := 0; i < 60 && hi.Sub(lo) > 10*time.Millisecond; i++ {
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

// bisectBool refines the change of f between lo and hi to under 10 ms.
func bisectBool(f func(time.Time) bool, lo, hi time.Time) time.Time {
	at := f(lo)
	for i := 0; i < 60 && hi.Sub(lo) > 10*time.Millisecond; i++ {
		mid := lo.Add(hi.Sub(lo) / 2)
		if f(mid) == at {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo.Add(hi.Sub(lo) / 2)
}

// ternaryMax returns the instant of greatest f in [lo, hi], f unimodal there.
func ternaryMax(f func(time.Time) float64, lo, hi time.Time) time.Time {
	for i := 0; i < 100 && hi.Sub(lo) > 10*time.Millisecond; i++ {
		third := hi.Sub(lo) / 3
		m1, m2 := lo.Add(third), hi.Add(-third)
		if f(m1) < f(m2) {
			lo = m1
		} else {
			hi = m2
		}
	}
	return lo.Add(hi.Sub(lo) / 2)
}
