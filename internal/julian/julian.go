// Package julian converts civil instants into the time quantities the ephemeris
// math depends on: the Julian Date, Greenwich and local mean sidereal time, and
// the mean obliquity of the ecliptic. It is a thin, correct wrapper over the
// meeus algorithms that keeps the rest of the library in plain float64 degrees.
package julian

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

	meeusjulian "github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/nutation"
	"github.com/soniakeys/meeus/v3/sidereal"

	"github.com/Bugs5382/go-astronomy/internal/angles"
)

// Date returns the Julian Date for the given instant. The instant is always
// interpreted in UTC, so a zoned time is converted before conversion.
func Date(t time.Time) float64 {
	return meeusjulian.TimeToJD(t.UTC())
}

// GreenwichSiderealTime returns the Greenwich mean sidereal time for the given
// instant, in degrees in the range [0, 360).
func GreenwichSiderealTime(t time.Time) float64 {
	// unit.Time.Hour is in [0, 24); scaling by 15 yields degrees.
	return angles.Normalize(sidereal.Mean(Date(t)).Hour() * 15)
}

// LocalSiderealTime returns the local mean sidereal time for an observer at the
// given east-positive longitude in degrees, as degrees in the range [0, 360).
func LocalSiderealTime(t time.Time, lonEastDeg float64) float64 {
	return angles.Normalize(GreenwichSiderealTime(t) + lonEastDeg)
}

// MeanObliquity returns the mean obliquity of the ecliptic for the given
// instant, in degrees, following the IAU 1980 polynomial.
func MeanObliquity(t time.Time) float64 {
	return nutation.MeanObliquity(Date(t)).Deg()
}
