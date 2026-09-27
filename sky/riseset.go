package sky

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
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// State says what a rise, set, or transit search found.
type State int

const (
	// Crossed means the event happens, at Event.Time.
	Crossed State = iota
	// AlwaysAbove means the target stays above the horizon for the whole
	// search window, so it neither rises nor sets: Earth from most of the
	// lunar near side, or the midnight Sun.
	AlwaysAbove
	// AlwaysBelow means the target stays below the horizon for the whole
	// search window: Earth from the lunar far side, or the polar night.
	AlwaysBelow
	// NoEvent means the window closed without the event while the target did
	// cross the horizon, or without a transit.
	NoEvent
)

// String returns a lowercase name for the state.
func (s State) String() string {
	switch s {
	case Crossed:
		return "crossed"
	case AlwaysAbove:
		return "always_above"
	case AlwaysBelow:
		return "always_below"
	default:
		return "no_event"
	}
}

// Event is the result of a rise, set, or transit search: the instant, in
// UTC, when State is Crossed, and the zero time otherwise.
type Event struct {
	Time  time.Time
	State State
}

// Found reports whether the event happens (State is Crossed).
func (e Event) Found() bool { return e.State == Crossed }

// Window returns how far ahead NextRise, NextSet, and NextTransit search
// from a site on body b: 30 days on Earth, as the v1 packages do, and
// elsewhere two solar days or two rotations, whichever is longer (65 days on
// the Moon, 2.3 sols on Mars).
func Window(b *Body) time.Duration {
	if b == nil {
		return 0
	}
	if b.kind == kindEarth {
		return 30 * 24 * time.Hour
	}
	return b.searchWindow()
}

// NextRise returns the first instant strictly after t at which the target's
// upper limb rises above the site's horizon: the horizon lowered by the
// body's horizontal refraction and by the dip of the site's height. A target
// that stays up or down for the whole Window reports AlwaysAbove or
// AlwaysBelow instead of failing.
//
// On Earth the Sun rises where earth.NewSunTimes starts its sunrise band, and
// the Moon and the planets rise where their own packages say. Off Earth the
// search steps at 1/96 of the shorter of the body's rotation and solar day
// and refines each crossing to under a second.
func NextRise(site Site, target *Body, t time.Time) (Event, error) {
	return next(site, target, t, eventRise)
}

// NextSet returns the first instant strictly after t at which the target's
// upper limb sets below the site's horizon, as NextRise.
func NextSet(site Site, target *Body, t time.Time) (Event, error) {
	return next(site, target, t, eventSet)
}

// NextTransit returns the first instant strictly after t of the target's
// upper culmination on the site's meridian, whether or not it is above the
// horizon then, or NoEvent when the target does not cross the meridian in
// the Window (Earth hangs near one place in the lunar sky and may not).
func NextTransit(site Site, target *Body, t time.Time) (Event, error) {
	return next(site, target, t, eventTransit)
}

type eventKind int

const (
	eventRise eventKind = iota
	eventSet
	eventTransit
)

func next(site Site, target *Body, t time.Time, kind eventKind) (Event, error) {
	if err := validate(site, target); err != nil {
		return Event{}, err
	}
	if site.Body.kind == kindEarth {
		return nextOnEarth(site, target, t, kind)
	}
	var f func(time.Time) float64
	if kind == eventTransit {
		f = func(when time.Time) float64 { return site.look(target, julian.TT(when)).hourAngle }
	} else {
		dip := site.dip()
		lift := site.Body.refraction
		f = func(when time.Time) float64 {
			l := site.look(target, julian.TT(when))
			return l.alt + l.semi + lift + dip
		}
	}
	return scan(f, t, site.Body.searchStep(), site.Body.searchWindow(), kind), nil
}

// dip is the geometric dip of the horizon, in degrees, for a site above its
// body's reference ellipsoid (off Earth, where no refraction bends it).
func (s Site) dip() float64 {
	h := s.Height.Meters() / 1000
	if !(h > 0) {
		return 0
	}
	return math.Acos(s.Body.model.EquatorialKm/(s.Body.model.EquatorialKm+h)) / radPerDeg
}

// scan steps f forward from t, looking for the sign change of the event,
// and refines it by bisection. For a rise or set, a window with no sign
// change at all is AlwaysAbove or AlwaysBelow. For the hour angle, the wrap
// from +180 to -180 is not a transit.
func scan(f func(time.Time) float64, t time.Time, step, window time.Duration, kind eventKind) Event {
	prev := f(t)
	first := prev
	changed := false
	for when := t.Add(step); !when.After(t.Add(window)); when = when.Add(step) {
		cur := f(when)
		up := prev < 0 && cur >= 0
		down := prev >= 0 && cur < 0
		if kind == eventTransit && math.Abs(cur-prev) > 180 {
			up, down = false, false
		}
		if up || down {
			changed = true
		}
		if (kind != eventSet && up) || (kind == eventSet && down) {
			return Event{Time: bisect(f, when.Add(-step), when).UTC(), State: Crossed}
		}
		prev = cur
	}
	switch {
	case kind == eventTransit || changed:
		return Event{State: NoEvent}
	case first >= 0:
		return Event{State: AlwaysAbove}
	default:
		return Event{State: AlwaysBelow}
	}
}

// bisect refines a sign change of f in [lo, hi] to under a second.
func bisect(f func(time.Time) float64, lo, hi time.Time) time.Time {
	fLo := f(lo)
	for i := 0; i < 80 && hi.Sub(lo) > time.Second; i++ {
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

// nextOnEarth answers from the v1 packages, so an Earth site gives their
// events exactly.
func nextOnEarth(site Site, target *Body, t time.Time, kind eventKind) (Event, error) {
	obs := site.observer()
	if target.kind == kindSun {
		return sunOnEarth(obs, t, kind)
	}
	var (
		when time.Time
		ok   bool
		err  error
	)
	switch {
	case target.kind == kindMoon && kind == eventRise:
		when, ok, err = moon.NextRise(obs, t)
	case target.kind == kindMoon && kind == eventSet:
		when, ok, err = moon.NextSet(obs, t)
	case target.kind == kindMoon:
		f := func(w time.Time) float64 { return site.look(target, julian.TT(w)).hourAngle }
		return scan(f, t, 10*time.Minute, 30*24*time.Hour, eventTransit), nil
	case kind == eventRise:
		when, ok, err = target.planet.NextRise(obs, t)
	case kind == eventSet:
		when, ok, err = target.planet.NextSet(obs, t)
	default:
		when, ok, err = target.planet.NextTransit(obs, t)
	}
	if err != nil {
		return Event{}, err
	}
	if ok {
		return Event{Time: when.UTC(), State: Crossed}, nil
	}
	if kind == eventTransit {
		return Event{State: NoEvent}, nil
	}
	// No crossing in 30 days: the side of the horizon it stayed on.
	res, err := Position(site, target, t)
	if err != nil {
		return Event{}, err
	}
	if res.Altitude+res.Diameter.Radius() > -earth.HorizonDip(site.Height) {
		return Event{State: AlwaysAbove}, nil
	}
	return Event{State: AlwaysBelow}, nil
}

// sunOnEarth finds the Sun's events in the civil days of earth.NewSunTimes:
// sunrise opens the sunrise band, sunset closes the sunset band, and transit
// is solar noon.
func sunOnEarth(obs astronomy.Observer, t time.Time, kind eventKind) (Event, error) {
	var polar earth.PolarState
	for d := 0; d <= 30; d++ {
		day, err := earth.NewSunTimes(obs, t.UTC().AddDate(0, 0, d))
		if err != nil {
			return Event{}, err
		}
		if d == 0 {
			polar, _ = day.Polar()
		}
		if kind == eventTransit {
			if noon, _ := day.SolarNoon(); noon.After(t) {
				return Event{Time: noon.UTC(), State: Crossed}, nil
			}
			continue
		}
		for _, s := range day.Segments() {
			var edge time.Time
			switch {
			case kind == eventRise && s.Label == earth.LabelSunrise:
				edge = s.From
			case kind == eventSet && s.Label == earth.LabelSunset:
				edge = s.To
			default:
				continue
			}
			// A band cut by midnight has an edge at the day boundary, which
			// is not the event.
			if edge.After(t) && !edge.Equal(day.DayStart()) && !edge.Equal(day.DayEnd()) {
				return Event{Time: edge.UTC(), State: Crossed}, nil
			}
		}
	}
	switch polar {
	case earth.MidnightSun:
		return Event{State: AlwaysAbove}, nil
	case earth.PolarNight:
		return Event{State: AlwaysBelow}, nil
	default:
		return Event{State: NoEvent}, nil
	}
}
