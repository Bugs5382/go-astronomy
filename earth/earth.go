// Package earth adds Earth-bound conventions on top of the universal solar
// geometry in the sun package: named twilight bands, sunrise and sunset,
// solar noon, polar states, and atmospheric refraction. The Sun itself is
// universal; the vocabulary of civil, nautical, and astronomical twilight,
// golden hour, and the -0.833 degree horizon crossing is an Earth-atmosphere
// convention that lives here.
//
// The altitude-to-band mapping is data, not a hardcoded rule: a Segmentation is
// a set of altitude thresholds with labels, and DefaultSegmentation ships the
// Earth defaults. A consumer may supply its own segmentation wholesale, for
// example to describe the same Sun with a different vocabulary.
//
// All altitudes are geometric center altitudes in degrees, matching the sun
// package. The horizon-crossing threshold of -0.833 degrees incorporates the
// standard allowance for atmospheric refraction and the Sun's semidiameter, so
// sunrise and sunset are found where the geometric center altitude crosses that
// value. The package is stateless and concurrency-safe: time is always a
// parameter, never captured.
package earth

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
)

// Standard band labels emitted by DefaultSegmentation. A consumer supplying a
// custom Segmentation is free to use any labels; these name the Earth defaults.
const (
	LabelNight            = "night"
	LabelAstronomicalDawn = "astronomical_dawn"
	LabelNauticalDawn     = "nautical_dawn"
	LabelCivilDawn        = "civil_dawn"
	LabelSunrise          = "sunrise"
	LabelGoldenHour       = "golden_hour"
	LabelDay              = "day"
	LabelSunset           = "sunset"
	LabelCivilDusk        = "civil_dusk"
	LabelNauticalDusk     = "nautical_dusk"
	LabelAstronomicalDusk = "astronomical_dusk"
)

// HorizonAltitude is the geometric center altitude, in degrees, at which the
// Sun's upper limb sits on the horizon under standard atmospheric refraction.
// It combines the mean solar semidiameter (about 16 arc minutes) with the
// horizontal refraction of about 34 arc minutes modeled by Refraction, and is
// the threshold used for sunrise and sunset.
const HorizonAltitude = -0.833

// DipArcminPerRootMetre is the coefficient of the horizon dip: the sea
// horizon seen from h metres lies 1.76 * sqrt(h) arc minutes below the
// astronomical horizon. It is the Nautical Almanac value, which includes
// standard terrestrial refraction, so it is the dip an observer actually sees
// and matches the refracted horizon the rise and set threshold already uses.
// The purely geometric dip, 1.93 * sqrt(h), ignores that refraction.
const DipArcminPerRootMetre = 1.76

// HorizonDip returns the dip of the sea horizon, in degrees, for an observer
// elevationM metres above sea level. It is zero at or below sea level, where
// no sea horizon lies below the observer.
//
// The dip assumes an unobstructed horizon at sea level; terrain that hides it
// makes the true horizon higher, not lower.
func HorizonDip(elevationM float64) float64 {
	if !(elevationM > 0) {
		return 0
	}
	return DipArcminPerRootMetre * math.Sqrt(elevationM) / 60
}

// Level is one altitude boundary in a Segmentation together with the labels of
// the band lying immediately above it. While the Sun climbs through the band
// (from solar midnight toward solar noon) the band is named Rising; while it
// sinks through it (from noon toward solar midnight) it is named Setting. The
// label follows the Sun's motion, not the clock, so a band that runs past local
// midnight keeps its label on both dates. The day is split at solar noon and at
// solar midnight, so the band above the highest Level has Rising (morning) and
// Setting (afternoon) halves.
type Level struct {
	// Altitude is the geometric center altitude of the boundary, in degrees.
	Altitude float64
	// Rising labels the band above this Level while the Sun climbs.
	Rising string
	// Setting labels the band above this Level while the Sun sinks.
	Setting string
	// DipCorrected lowers this Level by the horizon dip for the observer's
	// elevation (see HorizonDip). DefaultSegmentation sets it on the sunrise
	// and sunset levels only: by the USNO convention, civil, nautical, and
	// astronomical twilight are the Sun's depression below the astronomical
	// horizon and do not take the dip. WithTwilightDip sets it on every level.
	DipCorrected bool
}

// Segmentation divides the Sun's altitude over a civil day into named bands. It
// is a set of altitude Levels in ascending order; the band below the lowest
// Level is labeled Night, and each Level labels the band above it. Horizon is
// the altitude treated as the sunrise and sunset boundary and drives polar
// detection; it always takes the horizon dip for the observer's elevation.
type Segmentation struct {
	// Night labels the band below the lowest Level (deep night).
	Night string
	// Horizon is the altitude, in degrees, used for sunrise, sunset, and polar
	// state detection, at sea level. It is lowered by the horizon dip for an
	// observer above sea level.
	Horizon float64
	// Levels are the altitude boundaries in ascending order of Altitude.
	Levels []Level
}

// DefaultSegmentation is the Earth default division of the day: astronomical
// (-18), nautical (-12), and civil (-6) twilight, the sunrise and sunset
// horizon crossing (-0.833, upper limb including refraction), a short sunrise or
// sunset band up to -0.3, golden hour up to +6, and full day above that. It is
// a shared value; treat it as read-only and build a fresh Segmentation to
// customize.
//
// For an observer above sea level, the sunrise and sunset crossing and the top
// of the sunrise band are lowered by the horizon dip; the twilight levels are
// not (see Level.DipCorrected and WithTwilightDip).
var DefaultSegmentation = Segmentation{
	Night:   LabelNight,
	Horizon: HorizonAltitude,
	Levels: []Level{
		{Altitude: -18, Rising: LabelAstronomicalDawn, Setting: LabelAstronomicalDusk},
		{Altitude: -12, Rising: LabelNauticalDawn, Setting: LabelNauticalDusk},
		{Altitude: -6, Rising: LabelCivilDawn, Setting: LabelCivilDusk},
		{Altitude: HorizonAltitude, Rising: LabelSunrise, Setting: LabelSunset, DipCorrected: true},
		{Altitude: -0.3, Rising: LabelGoldenHour, Setting: LabelGoldenHour, DipCorrected: true},
		{Altitude: 6, Rising: LabelDay, Setting: LabelDay},
	},
}

// WithTwilightDip returns a copy of the segmentation with every level lowered
// by the horizon dip, so the twilight boundaries move with the observer's
// elevation along with sunrise and sunset. It departs from the USNO
// convention, under which twilight is measured from the astronomical horizon;
// use it when the twilight bands should track the horizon the observer
// actually sees. The receiver is not modified.
func (s Segmentation) WithTwilightDip() Segmentation {
	out := s
	out.Levels = make([]Level, len(s.Levels))
	for i, l := range s.Levels {
		l.DipCorrected = true
		out.Levels[i] = l
	}
	return out
}

// atElevation returns a copy of the segmentation for an observer elevationM
// metres above sea level: the horizon and every dip-corrected level lowered by
// the horizon dip. At sea level it returns the segmentation unchanged.
func (s Segmentation) atElevation(elevationM float64) Segmentation {
	dip := HorizonDip(elevationM)
	if dip == 0 {
		return s
	}
	out := s
	out.Horizon -= dip
	out.Levels = make([]Level, len(s.Levels))
	for i, l := range s.Levels {
		if l.DipCorrected {
			l.Altitude -= dip
		}
		out.Levels[i] = l
	}
	return out
}

// Refraction returns the atmospheric refraction, in degrees, that lifts a body
// seen at the given apparent altitude (degrees) above its true geometric
// altitude, using Bennett's formula for a standard atmosphere. The correction
// is about 0.57 degrees at the horizon and falls toward zero near the zenith.
// It is the model behind the -0.833 degree horizon crossing used for sunrise
// and sunset.
func Refraction(apparentAltDeg float64) float64 {
	// Bennett (1982): R = 1 / tan(h + 7.31/(h + 4.4)) arc minutes, with the
	// apparent altitude h in degrees.
	inner := apparentAltDeg + 7.31/(apparentAltDeg+4.4)
	arcMinutes := 1 / math.Tan(inner*math.Pi/180)
	return arcMinutes / 60
}

// PolarState describes a day on which the Sun does not cross the horizon.
type PolarState int

const (
	// NotPolar means the Sun both rises and sets during the civil day.
	NotPolar PolarState = iota
	// MidnightSun means the Sun stays above the horizon for the whole day.
	MidnightSun
	// PolarNight means the Sun stays below the horizon for the whole day.
	PolarNight
)

// String returns a lowercase name for the polar state.
func (p PolarState) String() string {
	switch p {
	case MidnightSun:
		return "midnight_sun"
	case PolarNight:
		return "polar_night"
	default:
		return "not_polar"
	}
}

// Segment is a resolved band of the day: a labeled, contiguous span of time.
// From is inclusive and To is exclusive, and Seconds is the span in seconds.
type Segment struct {
	Label   string
	From    time.Time
	To      time.Time
	Seconds float64
}

// Contains reports whether the instant t lies in the segment, with From
// inclusive and To exclusive.
func (s Segment) Contains(t time.Time) bool {
	return !t.Before(s.From) && t.Before(s.To)
}
