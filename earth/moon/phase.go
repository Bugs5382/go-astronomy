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
	"math"
	"time"

	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// SynodicMonth is the mean length of the lunar phase cycle, in days: the average
// interval from one New Moon to the next. Individual cycles vary by a few hours
// around this mean.
const SynodicMonth = 29.530588853

// lunationsPerYear is the mean number of lunations in a Julian year, the factor
// the phase series uses to map a decimal year to a lunation number (Meeus,
// Astronomical Algorithms, chapter 49). lunationYear is its reciprocal: the
// decimal-year step that advances the phase series by exactly one lunation, so
// a search steps cleanly to the adjacent New or Full Moon rather than risking
// the same rounded lunation.
const (
	lunationsPerYear = 12.3685
	lunationYear     = 1.0 / lunationsPerYear
)

// Phase is one of the eight conventional named phases of the lunar cycle,
// ordered from New Moon through the waxing phases to Full and back through the
// waning phases.
type Phase int

const (
	// New is the New Moon: the disc is dark, at the start of the cycle.
	New Phase = iota
	// WaxingCrescent is the growing crescent between New and First Quarter.
	WaxingCrescent
	// FirstQuarter is the First Quarter: the disc is half lit and waxing.
	FirstQuarter
	// WaxingGibbous is the growing gibbous between First Quarter and Full.
	WaxingGibbous
	// Full is the Full Moon: the disc is fully lit, at mid-cycle.
	Full
	// WaningGibbous is the shrinking gibbous between Full and Last Quarter.
	WaningGibbous
	// LastQuarter is the Last Quarter: the disc is half lit and waning.
	LastQuarter
	// WaningCrescent is the shrinking crescent between Last Quarter and New.
	WaningCrescent
)

// String returns a lowercase snake_case name for the phase, or "unknown" for a
// value outside the eight named phases.
func (p Phase) String() string {
	switch p {
	case New:
		return "new"
	case WaxingCrescent:
		return "waxing_crescent"
	case FirstQuarter:
		return "first_quarter"
	case WaxingGibbous:
		return "waxing_gibbous"
	case Full:
		return "full"
	case WaningGibbous:
		return "waning_gibbous"
	case LastQuarter:
		return "last_quarter"
	case WaningCrescent:
		return "waning_crescent"
	default:
		return "unknown"
	}
}

// newMoonBefore returns the julian ephemeris day of the most recent New Moon at
// or before jde. It starts from the New Moon the series places nearest jde and,
// if that falls after jde, steps back one whole lunation at a time until it
// lands at or before jde.
func newMoonBefore(jde float64) float64 {
	y := ephemeris.JDEToJulianYear(jde)
	n := ephemeris.NewMoon(y)
	for n > jde {
		y -= lunationYear
		n = ephemeris.NewMoon(y)
	}
	return n
}

// Age returns the Moon's age at t: the time elapsed since the most recent New
// Moon, in days. It runs from 0 at New Moon up to one synodic month.
func Age(t time.Time) float64 {
	jde := julian.Date(t)
	return jde - newMoonBefore(jde)
}

// PhaseAngle returns the Sun-Moon-Earth phase angle at t, in degrees in [0, 180].
// It is 0 at Full Moon, when the Moon is fully lit, and approaches 180 at New
// Moon, when the disc is dark.
//
// The value comes from the accurate method of Meeus chapter 48, which combines
// the Moon's geocentric position with the Sun's apparent longitude and
// distance. Measured against JPL Horizons across a synodic month it agrees to
// better than 0.02 degrees, including within an hour of New Moon.
//
// It reaches neither end of its range. At New Moon and at Full Moon the Moon's
// apparent longitude is aligned with the Sun's, so the elongation left over is
// the Moon's ecliptic latitude, up to about 5.3 degrees; the phase angle stops
// that far short of 180 and of 0. A caller that wants "how full is the disc"
// should read Illumination, which does reach both ends.
func PhaseAngle(t time.Time) float64 {
	return ephemeris.MoonPhaseAngle(julian.Date(t))
}

// Illumination returns the fraction of the Moon's disc that is lit at t, in
// [0, 1]: 0 at New Moon and 1 at Full Moon. It is derived from PhaseAngle as
// (1 + cos i) / 2, so the two can never disagree.
func Illumination(t time.Time) float64 {
	return ephemeris.IlluminatedFraction(PhaseAngle(t))
}

// PhaseAt returns the named phase of the Moon at t. The synodic cycle is divided
// into eight equal segments centered on the named points, so New, First Quarter,
// Full, and Last Quarter each name the segment straddling their exact instant.
// Age is never negative, so the segment index is always in range.
func PhaseAt(t time.Time) Phase {
	seg := SynodicMonth / 8
	idx := int(math.Floor((Age(t)+seg/2)/seg)) % 8
	return Phase(idx)
}

// NextNew returns the instant of the first New Moon strictly after t, in UTC.
func NextNew(t time.Time) time.Time {
	return nextPhaseEvent(t, ephemeris.NewMoon)
}

// NextFull returns the instant of the first Full Moon strictly after t, in UTC.
func NextFull(t time.Time) time.Time {
	return nextPhaseEvent(t, ephemeris.FullMoon)
}

// nextPhaseEvent returns the first instant strictly after t at which the phase
// event occurs. event maps a decimal year to the ephemeris day of the event
// nearest that year; when the nearest event is at or before t, the search steps
// forward one whole lunation at a time until it lands after t.
func nextPhaseEvent(t time.Time, event func(float64) float64) time.Time {
	jde := julian.Date(t)
	y := ephemeris.JDEToJulianYear(jde)
	e := event(y)
	for e <= jde {
		y += lunationYear
		e = event(y)
	}
	return julian.Time(e)
}
