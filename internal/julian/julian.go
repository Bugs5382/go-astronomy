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

	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

// Date returns the Julian Date for the given instant. The instant is always
// interpreted in UTC, so a zoned time is converted before conversion.
func Date(t time.Time) float64 {
	return ephemeris.TimeToJD(t)
}

// Time returns the UTC instant of a Julian Date. It is the inverse of Date.
func Time(jd float64) time.Time {
	return ephemeris.JDToTime(jd)
}

// JulianYear returns the Julian year corresponding to an instant, the epoch
// form the precession and lunar phase series take.
func JulianYear(t time.Time) float64 {
	return ephemeris.JDEToJulianYear(Date(t))
}

// GreenwichSiderealTime returns the Greenwich mean sidereal time for the given
// instant, in degrees in the range [0, 360).
func GreenwichSiderealTime(t time.Time) float64 {
	return ephemeris.MeanSiderealTime(Date(t))
}

// LocalSiderealTime returns the local mean sidereal time for an observer at the
// given east-positive longitude in degrees, as degrees in the range [0, 360).
func LocalSiderealTime(t time.Time, lonEastDeg float64) float64 {
	return angles.Normalize(GreenwichSiderealTime(t) + lonEastDeg)
}

// MeanObliquity returns the mean obliquity of the ecliptic for the given
// instant, in degrees, following the IAU 1980 polynomial.
func MeanObliquity(t time.Time) float64 {
	return ephemeris.MeanObliquity(Date(t))
}

// TT returns the Julian ephemeris day, on Terrestrial Time, of the UTC instant
// t. The ephemerides take this, not the UT Julian day (issue 45).
func TT(t time.Time) float64 {
	return ephemeris.TimeToJDE(t)
}

// FromTT returns the UTC instant of a Julian ephemeris day. It is the inverse
// of TT, for turning an event the ephemerides place in TT back into UTC.
func FromTT(jde float64) time.Time {
	return ephemeris.JDEToTime(jde)
}

// ApparentSiderealTime returns the Greenwich apparent sidereal time for the
// given instant, in degrees in [0, 360): mean sidereal time plus the nutation
// in right ascension. It is the sidereal time that matches positions referred
// to the true equator and equinox of date. UT1 is taken as UTC, which it
// tracks to within 0.9 s.
func ApparentSiderealTime(t time.Time) float64 {
	return angles.Normalize(GreenwichSiderealTime(t) + ephemeris.NutationInRA(TT(t)))
}
