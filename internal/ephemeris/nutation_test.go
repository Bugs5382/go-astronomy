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

// dms converts degrees, arc minutes, and arc seconds to decimal degrees.
func dms(d, m, s float64) float64 { return d + m/60 + s/3600 }

// hms converts hours, minutes, and seconds of time to decimal hours.
func hms(h, m, s float64) float64 { return h + m/60 + s/3600 }

// TestNutation anchors nutation in longitude and obliquity against Meeus,
// Astronomical Algorithms, example 22.a: for JDE 2446895.5 the book gives
// nutation in longitude -3.788 arc seconds and nutation in obliquity
// +9.443 arc seconds.
func TestNutation(t *testing.T) {
	t.Parallel()
	dpsi, deps := Nutation(2446895.5)
	if got, want := dpsi*3600, -3.788; math.Abs(got-want) > 0.001 {
		t.Errorf("nutation in longitude = %.4f arcsec, want %.3f", got, want)
	}
	if got, want := deps*3600, 9.443; math.Abs(got-want) > 0.001 {
		t.Errorf("nutation in obliquity = %.4f arcsec, want %.3f", got, want)
	}
}

// TestNutationIsPeriodic checks the leading 18.6 year term dominates: nutation
// in longitude stays inside its physical envelope of about 17.2 arc seconds
// plus the smaller terms, and changes sign over a nodal period.
func TestNutationIsPeriodic(t *testing.T) {
	t.Parallel()
	var sawPositive, sawNegative bool
	for i := 0; i < 200; i++ {
		jde := 2446895.5 + float64(i)*40
		dpsi, deps := Nutation(jde)
		if math.Abs(dpsi*3600) > 20 {
			t.Errorf("nutation in longitude %.3f arcsec at JDE %v, out of envelope", dpsi*3600, jde)
		}
		if math.Abs(deps*3600) > 10 {
			t.Errorf("nutation in obliquity %.3f arcsec at JDE %v, out of envelope", deps*3600, jde)
		}
		if dpsi > 0 {
			sawPositive = true
		} else {
			sawNegative = true
		}
	}
	if !sawPositive || !sawNegative {
		t.Error("nutation in longitude never changed sign over the sampled span")
	}
}

// TestMeanObliquity anchors the mean obliquity of the ecliptic against Meeus
// example 22.a, which gives 23d26'27.407" for JDE 2446895.5, and against
// example 25.a, which gives 23d26'24.83" for JDE 2448908.5.
func TestMeanObliquity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		jde  float64
		want float64
	}{
		{2446895.5, dms(23, 26, 27.407)},
		{2448908.5, dms(23, 26, 24.83)},
		{J2000, 23.4392911}, // the standard value at the epoch
	}
	for _, c := range cases {
		if got := MeanObliquity(c.jde); math.Abs(got-c.want) > 0.5/3600 {
			t.Errorf("MeanObliquity(%v) = %.9f deg, want %.9f deg (%.4f arcsec off)",
				c.jde, got, c.want, (got-c.want)*3600)
		}
	}
}

// TestTrueObliquity anchors the true obliquity against Meeus example 22.a,
// which gives 23d26'36.850" for JDE 2446895.5.
func TestTrueObliquity(t *testing.T) {
	t.Parallel()
	want := dms(23, 26, 36.850)
	if got := TrueObliquity(2446895.5); math.Abs(got-want) > 0.001/3600 {
		t.Errorf("TrueObliquity = %.9f deg, want %.9f deg (%.4f arcsec off)",
			got, want, (got-want)*3600)
	}
}

// TestNutationInRA anchors the equation of the equinoxes against the difference
// between mean and apparent sidereal time in Meeus example 12.a: mean
// 13h10m46.3668s and apparent 13h10m46.1351s, a difference of -0.2317 seconds
// of time.
func TestNutationInRA(t *testing.T) {
	t.Parallel()
	// Degrees to seconds of time: 1 degree is 240 seconds.
	got := NutationInRA(2446895.5) * 240
	want := hms(13, 10, 46.1351) - hms(13, 10, 46.3668)
	want *= 3600
	if math.Abs(got-want) > 0.001 {
		t.Errorf("NutationInRA = %.6f s, want %.6f s", got, want)
	}
}

// TestMeanSiderealTime anchors mean sidereal time at Greenwich against Meeus
// examples 12.a (13h10m46.3668s at JD 2446895.5) and 12.b (8h34m57.0896s at
// JD 2446896.30625).
func TestMeanSiderealTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		jd   float64
		want float64 // hours
	}{
		{2446895.5, hms(13, 10, 46.3668)},
		{2446896.30625, hms(8, 34, 57.0896)},
	}
	for _, c := range cases {
		got := MeanSiderealTime(c.jd)
		if got < 0 || got >= 360 {
			t.Errorf("MeanSiderealTime(%v) = %v, out of [0,360)", c.jd, got)
		}
		// A ten-thousandth of a second of time, the precision the book quotes.
		if math.Abs(got/15-c.want) > 0.0001/3600 {
			t.Errorf("MeanSiderealTime(%v) = %.9f h, want %.9f h", c.jd, got/15, c.want)
		}
	}
}

// TestApparentSiderealTime anchors apparent sidereal time at Greenwich against
// Meeus example 12.a, which gives 13h10m46.1351s at JD 2446895.5.
func TestApparentSiderealTime(t *testing.T) {
	t.Parallel()
	got := ApparentSiderealTime(2446895.5)
	if got < 0 || got >= 360 {
		t.Errorf("ApparentSiderealTime = %v, out of [0,360)", got)
	}
	want := hms(13, 10, 46.1351)
	if math.Abs(got/15-want) > 0.0001/3600 {
		t.Errorf("ApparentSiderealTime = %.9f h, want %.9f h", got/15, want)
	}
}

// TestSiderealTimeAdvancesFasterThanSolar checks the defining property of
// sidereal time: it gains about 3 minutes 56 seconds on the solar day.
func TestSiderealTimeAdvancesFasterThanSolar(t *testing.T) {
	t.Parallel()
	a := MeanSiderealTime(2451545.0)
	b := MeanSiderealTime(2451546.0)
	gain := b - a
	if gain < 0 {
		gain += 360
	}
	// 3m56.55s of time is 0.9856 degrees.
	if math.Abs(gain-0.9856) > 0.001 {
		t.Errorf("sidereal gain over one day = %.6f deg, want ~0.9856", gain)
	}
}

func TestPMod(t *testing.T) {
	t.Parallel()
	cases := []struct{ x, y, want float64 }{
		{7, 3, 1},
		{-1, 3, 2},
		{-7, 3, 2},
		{0, 3, 0},
		{3, 3, 0},
		{-86400, 86400, 0},
	}
	for _, c := range cases {
		if got := pmod(c.x, c.y); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("pmod(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}
