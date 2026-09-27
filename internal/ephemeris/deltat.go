package ephemeris

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

// Time scales: Terrestrial Time against UTC.
//
// The ephemerides in this package run on Terrestrial Time (TT), the uniform
// time scale of the theories, while a Go time.Time is UTC. The two differ by
// TT - UTC, which has been 69.184 seconds since 2017 (issue 45).
//
// From 1972 the offset is exact: TT = TAI + 32.184 s by definition, and
// TAI - UTC is the integer count of leap seconds published in IERS Bulletin C.
// A time.Time never holds a leap second, so the table lookup is exact on every
// representable instant. Before 1972 UTC was not a leap-second scale; there the
// offset is taken as Delta-T (TT - UT1) from the polynomial fits of Espenak and
// Meeus, "Five Millennium Canon of Solar Eclipses", NASA/TP-2006-214141, which
// treat UTC as UT1.
//
// After the last table entry the offset is held. The CGPM resolved in 2022 to
// stop inserting leap seconds by 2035, and until then any new one is announced
// six months ahead in Bulletin C; holding the value costs at most the
// |UT1 - UTC| <= 0.9 s that the leap seconds exist to bound.

import "time"

// ttMinusTAI is the fixed offset TT - TAI, in seconds.
const ttMinusTAI = 32.184

// leapSeconds lists each date from which TAI - UTC took a new value, from IERS
// Bulletin C (https://hpiers.obspm.fr/iers/bul/bulc/Leap_Second.dat). The
// table was checked against Bulletin C in September 2026, which announced no
// leap second after 2017 January 1.
var leapSeconds = []struct {
	from     time.Time
	taiMinus float64
}{
	{time.Date(1972, 1, 1, 0, 0, 0, 0, time.UTC), 10},
	{time.Date(1972, 7, 1, 0, 0, 0, 0, time.UTC), 11},
	{time.Date(1973, 1, 1, 0, 0, 0, 0, time.UTC), 12},
	{time.Date(1974, 1, 1, 0, 0, 0, 0, time.UTC), 13},
	{time.Date(1975, 1, 1, 0, 0, 0, 0, time.UTC), 14},
	{time.Date(1976, 1, 1, 0, 0, 0, 0, time.UTC), 15},
	{time.Date(1977, 1, 1, 0, 0, 0, 0, time.UTC), 16},
	{time.Date(1978, 1, 1, 0, 0, 0, 0, time.UTC), 17},
	{time.Date(1979, 1, 1, 0, 0, 0, 0, time.UTC), 18},
	{time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), 19},
	{time.Date(1981, 7, 1, 0, 0, 0, 0, time.UTC), 20},
	{time.Date(1982, 7, 1, 0, 0, 0, 0, time.UTC), 21},
	{time.Date(1983, 7, 1, 0, 0, 0, 0, time.UTC), 22},
	{time.Date(1985, 7, 1, 0, 0, 0, 0, time.UTC), 23},
	{time.Date(1988, 1, 1, 0, 0, 0, 0, time.UTC), 24},
	{time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), 25},
	{time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC), 26},
	{time.Date(1992, 7, 1, 0, 0, 0, 0, time.UTC), 27},
	{time.Date(1993, 7, 1, 0, 0, 0, 0, time.UTC), 28},
	{time.Date(1994, 7, 1, 0, 0, 0, 0, time.UTC), 29},
	{time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC), 30},
	{time.Date(1997, 7, 1, 0, 0, 0, 0, time.UTC), 31},
	{time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC), 32},
	{time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC), 33},
	{time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC), 34},
	{time.Date(2012, 7, 1, 0, 0, 0, 0, time.UTC), 35},
	{time.Date(2015, 7, 1, 0, 0, 0, 0, time.UTC), 36},
	{time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC), 37},
}

// TTMinusUTC returns TT - UTC, in seconds, at the UTC instant t.
func TTMinusUTC(t time.Time) float64 {
	t = t.UTC()
	if t.Before(leapSeconds[0].from) {
		return deltaTPolynomial(t)
	}
	tai := leapSeconds[0].taiMinus
	for _, ls := range leapSeconds[1:] {
		if t.Before(ls.from) {
			break
		}
		tai = ls.taiMinus
	}
	return ttMinusTAI + tai
}

// deltaTPolynomial returns Delta-T in seconds from the Espenak and Meeus (2006)
// piecewise polynomials, evaluated at the decimal year of t.
func deltaTPolynomial(t time.Time) float64 {
	y := float64(t.Year()) + (float64(t.YearDay())-0.5)/daysInYear(t.Year())
	switch {
	case y < -500:
		u := (y - 1820) / 100
		return -20 + 32*u*u
	case y < 500:
		return Horner(y/100, 10583.6, -1014.41, 33.78311, -5.952053, -0.1798452, 0.022174192, 0.0090316521)
	case y < 1600:
		return Horner((y-1000)/100, 1574.2, -556.01, 71.23472, 0.319781, -0.8503463, -0.005050998, 0.0083572073)
	case y < 1700:
		return Horner(y-1600, 120, -0.9808, -0.01532, 1.0/7129)
	case y < 1800:
		return Horner(y-1700, 8.83, 0.1603, -0.0059285, 0.00013336, -1.0/1174000)
	case y < 1860:
		return Horner(y-1800, 13.72, -0.332447, 0.0068612, 0.0041116, -0.00037436, 0.0000121272, -0.0000001699, 0.000000000875)
	case y < 1900:
		return Horner(y-1860, 7.62, 0.5737, -0.251754, 0.01680668, -0.0004473624, 1.0/233174)
	case y < 1920:
		return Horner(y-1900, -2.79, 1.494119, -0.0598939, 0.0061966, -0.000197)
	case y < 1941:
		return Horner(y-1920, 21.20, 0.84493, -0.076100, 0.0020936)
	case y < 1961:
		return Horner(y-1950, 29.07, 0.407, -1.0/233, 1.0/2547)
	default:
		return Horner(y-1975, 45.45, 1.067, -1.0/260, -1.0/718)
	}
}

func daysInYear(y int) float64 {
	if time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC).YearDay() == 366 {
		return 366
	}
	return 365
}

// TimeToJDE returns the Julian ephemeris day (TT) of the UTC instant t.
func TimeToJDE(t time.Time) float64 {
	return TimeToJD(t) + TTMinusUTC(t)/86400
}

// JDEToTime returns the UTC instant of a Julian ephemeris day (TT). It is the
// inverse of TimeToJDE. The offset depends on the UTC instant being solved
// for, so it is found by fixed-point iteration, which settles in two steps
// because the offset changes by at most a second across a leap.
func JDEToTime(jde float64) time.Time {
	utc := JDToTime(jde)
	for i := 0; i < 3; i++ {
		next := JDToTime(jde - TTMinusUTC(utc)/86400)
		if next.Equal(utc) {
			break
		}
		utc = next
	}
	return utc
}
