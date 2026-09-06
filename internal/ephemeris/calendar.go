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

// Calendar and Julian day conversions.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 7,
// "Julian day": formula (7.1) for the forward conversion and the algorithm on
// p. 63 for the inverse. The Go form derives from soniakeys/meeus (julian),
// MIT licensed.
//
// Only the Gregorian calendar is implemented. Go's time.Time is proleptic
// Gregorian, so it is the only calendar this library can round-trip, and the
// Julian calendar branch of the book's algorithm would be unreachable.

import (
	"math"
	"time"
)

// GregorianToJD returns the Julian day of a date in the proleptic Gregorian
// calendar, with d the day of the month including its fractional part
// (Meeus 7.1).
//
// Negative years are accepted, back to JD 0. Results before JD 0 are not
// meaningful.
func GregorianToJD(y, m int, d float64) float64 {
	// January and February are counted as months 13 and 14 of the preceding
	// year, which is what makes the leap-day correction a single expression.
	if m <= 2 {
		y--
		m += 12
	}
	a := FloorDiv(y, 100)
	b := 2 - a + FloorDiv(a, 4)
	// 36525*(y+4716)/100 is INT(365.25*(y+4716)) without floating-point
	// rounding, and needs 64 bits because the product exceeds a 32-bit int.
	return float64(FloorDiv64(36525*int64(y+4716), 100)) +
		float64(FloorDiv(306*(m+1), 10)+b) + d - 1524.5
}

// JDToGregorian returns the proleptic Gregorian calendar date of a Julian day,
// with day carrying its fractional part (Meeus, chapter 7, p. 63).
//
// The date is Gregorian even for Julian days that precede the Gregorian reform,
// which is what Go's time.Time requires.
func JDToGregorian(jd float64) (year, month int, day float64) {
	zf, f := math.Modf(jd + .5)
	z := int64(zf)
	alpha := FloorDiv64(z*100-186721625, 3652425)
	a := z + 1 + alpha - FloorDiv64(alpha, 4)
	b := a + 1524
	c := FloorDiv64(b*100-12210, 36525)
	d := FloorDiv64(36525*c, 100)
	e := int(FloorDiv64((b-d)*1e4, 306001))

	day = float64(int(b-d)-FloorDiv(306001*e, 1e4)) + f
	switch e {
	case 14, 15:
		month = e - 13
	default:
		month = e - 1
	}
	switch month {
	case 1, 2:
		year = int(c) - 4715
	default:
		year = int(c) - 4716
	}
	return
}

// TimeToJD returns the Julian day of an instant. The instant is interpreted in
// UTC, so any zone offset is applied before conversion rather than ignored.
func TimeToJD(t time.Time) float64 {
	ut := t.UTC()
	y, m, _ := ut.Date()
	// Day zero of a month is the last day of the previous one, so this duration
	// is the day of the month including its time-of-day fraction.
	d := ut.Sub(time.Date(y, m, 0, 0, 0, 0, 0, time.UTC))
	return GregorianToJD(y, int(m), float64(d)/float64(24*time.Hour))
}

// JDToTime returns the UTC instant of a Julian day.
func JDToTime(jd float64) time.Time {
	y, m, d := JDToGregorian(jd)
	// Start from day zero of the month and add the fractional day back, which
	// keeps the sub-second part of d instead of truncating it to a date.
	t := time.Date(y, time.Month(m), 0, 0, 0, 0, 0, time.UTC)
	return t.Add(time.Duration(d * 24 * float64(time.Hour)))
}
