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

// Level is one altitude boundary in a Segmentation together with the labels of
// the band lying immediately above it. As the Sun climbs above Altitude toward
// solar noon it enters the band named Rising; as it sinks below Altitude after
// noon it enters the band named Setting. The band above the highest Level is
// the daytime band and is split at solar noon into its Rising (morning) and
// Setting (afternoon) halves.
type Level struct {
	// Altitude is the geometric center altitude of the boundary, in degrees.
	Altitude float64
	// Rising labels the band above this Level on the morning side of noon.
	Rising string
	// Setting labels the band above this Level on the evening side of noon.
	Setting string
}

// Segmentation divides the Sun's altitude over a civil day into named bands. It
// is a set of altitude Levels in ascending order; the band below the lowest
// Level is labeled Night, and each Level labels the band above it. Horizon is
// the altitude treated as the sunrise and sunset boundary and drives polar
// detection.
type Segmentation struct {
	// Night labels the band below the lowest Level (deep night).
	Night string
	// Horizon is the altitude, in degrees, used for sunrise, sunset, and polar
	// state detection.
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
var DefaultSegmentation = Segmentation{
	Night:   LabelNight,
	Horizon: HorizonAltitude,
	Levels: []Level{
		{Altitude: -18, Rising: LabelAstronomicalDawn, Setting: LabelAstronomicalDusk},
		{Altitude: -12, Rising: LabelNauticalDawn, Setting: LabelNauticalDusk},
		{Altitude: -6, Rising: LabelCivilDawn, Setting: LabelCivilDusk},
		{Altitude: HorizonAltitude, Rising: LabelSunrise, Setting: LabelSunset},
		{Altitude: -0.3, Rising: LabelGoldenHour, Setting: LabelGoldenHour},
		{Altitude: 6, Rising: LabelDay, Setting: LabelDay},
	},
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
