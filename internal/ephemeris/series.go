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

// Series evaluation, integer floor division, and the epoch constants shared by
// every algorithm in this package.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed.: the "General
// remarks" on the INT function in chapter 7, the definition of T in chapter 22
// (22.1), and the Julian and Besselian year definitions in chapter 21.
// The Go form derives from soniakeys/meeus (base), MIT licensed.

// Epochs and periods, in days.
const (
	// J2000 is the Julian date of the standard epoch J2000.0, 2000 January 1.5
	// TT.
	J2000 = 2451545.0
	// JulianYear is the length of a Julian year.
	JulianYear = 365.25
	// JulianCentury is the length of a Julian century, the unit of T.
	JulianCentury = 36525
)

// Horner evaluates the polynomial with coefficients c at x, with c[0] the
// constant term, using Horner's method. An empty coefficient list is the zero
// polynomial.
//
// The book writes its series as ascending polynomials in T, so keeping the
// constant term first lets each series be transcribed in the order it is
// printed.
func Horner(x float64, c ...float64) float64 {
	if len(c) == 0 {
		return 0
	}
	i := len(c) - 1
	y := c[i]
	for i > 0 {
		i--
		y = y*x + c[i]
	}
	return y
}

// FloorDiv returns the floor of x/y using integer arithmetic only. It stands in
// for the book's INT function where INT means "round toward minus infinity",
// which is what Go's truncating division does not do for negative operands. It
// panics when y is zero, as integer division does.
func FloorDiv(x, y int) int {
	q := x / y
	if (x < 0) != (y < 0) && x%y != 0 {
		q--
	}
	return q
}

// FloorDiv64 is FloorDiv for int64, needed where an intermediate product
// overflows a 32-bit int on platforms where int is 32 bits.
func FloorDiv64(x, y int64) int64 {
	q := x / y
	if (x < 0) != (y < 0) && x%y != 0 {
		q--
	}
	return q
}

// J2000Century returns T, the number of Julian centuries elapsed since J2000.0
// at the given Julian ephemeris day. T is the argument of nearly every series
// in this package (Meeus 22.1).
func J2000Century(jde float64) float64 {
	return (jde - J2000) / JulianCentury
}

// JDEToJulianYear returns the Julian year corresponding to a Julian ephemeris
// day. Precession epochs and the lunar phase search are expressed in Julian
// years rather than days.
func JDEToJulianYear(jde float64) float64 {
	return 2000 + (jde-J2000)/JulianYear
}

// JulianYearToJDE returns the Julian ephemeris day of a Julian year. It is the
// inverse of JDEToJulianYear.
func JulianYearToJDE(jy float64) float64 {
	return J2000 + JulianYear*(jy-2000)
}
