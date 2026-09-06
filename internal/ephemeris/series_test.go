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

import (
	"math"
	"testing"
)

func TestHorner(t *testing.T) {
	t.Parallel()
	// 2 + 3x + 4x^2 at x = 5 is 2 + 15 + 100 = 117.
	if got := Horner(5, 2, 3, 4); got != 117 {
		t.Errorf("Horner(5, 2, 3, 4) = %v, want 117", got)
	}
	// A single coefficient is a constant.
	if got := Horner(99, 7); got != 7 {
		t.Errorf("Horner(99, 7) = %v, want 7", got)
	}
	// The empty coefficient list is the zero polynomial.
	if got := Horner(3); got != 0 {
		t.Errorf("Horner(3) = %v, want 0", got)
	}
}

// TestHornerMatchesDirectEvaluation guards the coefficient order: Horner takes
// the constant term first, and reversing it would still produce a plausible
// number.
func TestHornerMatchesDirectEvaluation(t *testing.T) {
	t.Parallel()
	c := []float64{1.5, -2.25, 0.5, 3}
	for _, x := range []float64{-2, -0.1, 0, 0.25, 7} {
		want := c[0] + c[1]*x + c[2]*x*x + c[3]*x*x*x
		if got := Horner(x, c...); math.Abs(got-want) > 1e-12 {
			t.Errorf("Horner(%v) = %v, want %v", x, got, want)
		}
	}
}

func TestFloorDiv(t *testing.T) {
	t.Parallel()
	cases := []struct{ x, y, want int }{
		{7, 2, 3},
		{-7, 2, -4}, // floor(-3.5) is -4, not -3 as Go's / gives
		{7, -2, -4},
		{-7, -2, 3},
		{8, 2, 4},
		{-8, 2, -4}, // exact division needs no adjustment
		{0, 5, 0},
	}
	for _, c := range cases {
		if got := FloorDiv(c.x, c.y); got != c.want {
			t.Errorf("FloorDiv(%d, %d) = %d, want %d", c.x, c.y, got, c.want)
		}
		want64 := int64(c.want)
		if got := FloorDiv64(int64(c.x), int64(c.y)); got != want64 {
			t.Errorf("FloorDiv64(%d, %d) = %d, want %d", c.x, c.y, got, want64)
		}
	}
}

func TestEpochConstants(t *testing.T) {
	t.Parallel()
	if J2000 != 2451545.0 {
		t.Errorf("J2000 = %v, want 2451545.0", J2000)
	}
	if JulianCentury != 36525 {
		t.Errorf("JulianCentury = %v, want 36525", JulianCentury)
	}
	if JulianYear != 365.25 {
		t.Errorf("JulianYear = %v, want 365.25", JulianYear)
	}
}

// TestJ2000Century anchors T against Meeus example 25.a (JDE 2448908.5, for
// which the book gives T = -0.072183436) and example 22.a (JDE 2446895.5, for
// which T = -0.127296372).
func TestJ2000Century(t *testing.T) {
	t.Parallel()
	cases := []struct{ jde, want float64 }{
		{2448908.5, -0.072183436},
		{2446895.5, -0.127296372},
		{J2000, 0},
	}
	for _, c := range cases {
		if got := J2000Century(c.jde); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("J2000Century(%v) = %.12f, want %.9f", c.jde, got, c.want)
		}
	}
}

// TestJulianYearRoundTrip checks the Julian-year conversions used by the
// precession and lunar-phase searches invert each other exactly at the epoch
// and to floating-point accuracy elsewhere.
func TestJulianYearRoundTrip(t *testing.T) {
	t.Parallel()
	if got := JDEToJulianYear(J2000); got != 2000 {
		t.Errorf("JDEToJulianYear(J2000) = %v, want 2000", got)
	}
	if got := JulianYearToJDE(2000); got != J2000 {
		t.Errorf("JulianYearToJDE(2000) = %v, want %v", got, J2000)
	}
	for _, jde := range []float64{2415020.0, 2448908.5, 2462088.69} {
		back := JulianYearToJDE(JDEToJulianYear(jde))
		if math.Abs(back-jde) > 1e-6 {
			t.Errorf("round trip %v -> %v", jde, back)
		}
	}
	// Meeus example 21.b precesses to the equinox of JDE 2462088.69, which the
	// book identifies as the Julian year 2028.867050.
	if got := JDEToJulianYear(2462088.69); math.Abs(got-2028.867050) > 1e-6 {
		t.Errorf("JDEToJulianYear(2462088.69) = %.6f, want 2028.867050", got)
	}
}
