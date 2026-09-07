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

// example25a is the instant of Meeus, Astronomical Algorithms, example 25.a:
// 1992 October 13.0 TD, for which the book gives T = -0.072183436.
const example25a = 2448908.5

// TestSolarMeanAnomaly anchors the Earth's mean anomaly against example 25.a,
// which gives M = 278.99397 degrees.
func TestSolarMeanAnomaly(t *testing.T) {
	t.Parallel()
	got := pmod(SolarMeanAnomaly(J2000Century(example25a)), 360)
	if math.Abs(got-278.99397) > 1e-5 {
		t.Errorf("SolarMeanAnomaly = %.7f, want 278.99397", got)
	}
}

// TestSolarTrueLongitude anchors the Sun's true geometric longitude and true
// anomaly against example 25.a, which gives a geometric mean longitude of
// 201.807193 degrees, an equation of the center of -1.89732 degrees, and so a
// true longitude of 199.90988 degrees. The true anomaly is the mean anomaly
// plus the same equation of the center, 277.09665 degrees.
func TestSolarTrueLongitude(t *testing.T) {
	t.Parallel()
	tc := J2000Century(example25a)
	lon, anomaly := SolarTrueLongitude(tc)
	if math.Abs(lon-199.90988) > 1e-5 {
		t.Errorf("true longitude = %.7f, want 199.90988", lon)
	}
	if math.Abs(anomaly-277.09665) > 1e-5 {
		t.Errorf("true anomaly = %.7f, want 277.09665", anomaly)
	}
	// The equation of the center is the difference from the mean, in both.
	c1 := lon - pmod(SolarMeanLongitude(tc), 360)
	c2 := anomaly - pmod(SolarMeanAnomaly(tc), 360)
	if math.Abs(c1-c2) > 1e-9 {
		t.Errorf("equation of the center differs: %.9f vs %.9f", c1, c2)
	}
	if math.Abs(c1-(-1.89732)) > 1e-5 {
		t.Errorf("equation of the center = %.7f, want -1.89732", c1)
	}
}

// TestSolarMeanLongitude anchors the geometric mean longitude against example
// 25.a, which gives L0 = 201.807193 degrees.
func TestSolarMeanLongitude(t *testing.T) {
	t.Parallel()
	got := pmod(SolarMeanLongitude(J2000Century(example25a)), 360)
	if math.Abs(got-201.807193) > 1e-5 {
		t.Errorf("SolarMeanLongitude = %.7f, want 201.807193", got)
	}
}

// TestSolarEccentricity checks the eccentricity of the Earth's orbit against
// example 25.a, about 0.0167117, and against its slow secular decrease: it was
// larger a century ago and will be smaller a century hence.
func TestSolarEccentricity(t *testing.T) {
	t.Parallel()
	if got := SolarEccentricity(J2000Century(example25a)); math.Abs(got-0.0167117) > 1e-6 {
		t.Errorf("SolarEccentricity = %.9f, want ~0.0167117", got)
	}
	past := SolarEccentricity(-1)
	now := SolarEccentricity(0)
	future := SolarEccentricity(1)
	if !(past > now && now > future) {
		t.Errorf("eccentricity is not decreasing: %v, %v, %v", past, now, future)
	}
}

// TestSolarRadius anchors the Sun-Earth distance against example 25.a, which
// gives R = 0.99766 AU, and checks the annual range brackets perihelion and
// aphelion.
func TestSolarRadius(t *testing.T) {
	t.Parallel()
	if got := SolarRadius(J2000Century(example25a)); math.Abs(got-0.99766) > 1e-5 {
		t.Errorf("SolarRadius = %.7f, want 0.99766", got)
	}
	// Sample a year at three day steps: the distance must stay inside the
	// perihelion and aphelion bounds and must reach close to both.
	var lo, hi = math.Inf(1), math.Inf(-1)
	for i := 0; i < 122; i++ {
		r := SolarRadius(J2000Century(J2000 + float64(i)*3))
		lo = math.Min(lo, r)
		hi = math.Max(hi, r)
	}
	if lo < 0.98 || lo > 0.9834 {
		t.Errorf("minimum radius %.6f AU is not near perihelion", lo)
	}
	if hi < 1.0166 || hi > 1.02 {
		t.Errorf("maximum radius %.6f AU is not near aphelion", hi)
	}
}

// TestSolarApparentLongitude anchors the apparent longitude, which adds the
// corrections for nutation and aberration, against example 25.a: 199.90895
// degrees.
func TestSolarApparentLongitude(t *testing.T) {
	t.Parallel()
	got := SolarApparentLongitude(J2000Century(example25a))
	if math.Abs(got-199.90895) > 1e-5 {
		t.Errorf("SolarApparentLongitude = %.7f, want 199.90895", got)
	}
}

// TestSolarApparentEquatorial anchors the Sun's apparent equatorial position
// against the low-accuracy result of example 25.a: right ascension
// 13h13m31.4s and declination -7d47'06".
func TestSolarApparentEquatorial(t *testing.T) {
	t.Parallel()
	ra, dec := SolarApparentEquatorial(example25a)
	wantRA := hoursToDeg(13, 13, 31.4)
	wantDec := -dms(7, 47, 6)
	// A tenth of a second of right ascension is 1/2400 degree; an arc second of
	// declination is 1/3600 degree. Both are the precision the book prints.
	if math.Abs(ra-wantRA) > 1.0/2400 {
		t.Errorf("apparent RA = %.7f, want %.7f", ra, wantRA)
	}
	if math.Abs(dec-wantDec) > 1.0/3600 {
		t.Errorf("apparent Dec = %.7f, want %.7f", dec, wantDec)
	}
	if ra < 0 || ra >= 360 {
		t.Errorf("apparent RA = %v out of [0,360)", ra)
	}
}

// ephemerisApparentSun is the Sun's apparent geocentric right ascension and
// declination in degrees, and its geocentric distance in astronomical units,
// from the JPL Horizons system (target 10, center 500@399, referred to the true
// equator and equinox of date). These come from outside this library and outside
// the book.
var ephemerisApparentSun = []struct {
	jd, ra, dec, distance float64
}{
	{2445093.166666667, 40.43755, 15.70779, 1.00818792359644},   // 1982-05-03 16:00 UT
	{2448908.5, 198.37875, -7.78407, 0.99760832548205},          // 1992-10-13 00:00 UT
	{2461288.020833333, 163.40180, 7.05982, 1.00847421895488},   // 2026-09-04 12:30 UT
	{2461396.208333333, 269.82259, -23.43732, 0.98374267926946}, // 2026-12-21 17:00 UT
}

// TestSolarApparentEquatorialAgainstEphemeris measures the low-accuracy solar
// series against a modern numerical ephemeris.
//
// Meeus states the accuracy of chapter 25 as about 0.01 degrees in longitude.
// Two further sources of disagreement are expected here and are deliberate
// choices of this library: it does not model Delta-T, so a Julian day derived
// from UTC is treated as if it were dynamical time, and the series omits the
// higher-order aberration term. Agreement within 0.01 degrees of right
// ascension is therefore the correct expectation, not a defect, and it is far
// inside the arcminute-class accuracy this library documents.
func TestSolarApparentEquatorialAgainstEphemeris(t *testing.T) {
	t.Parallel()
	const tol = 0.01
	for _, e := range ephemerisApparentSun {
		ra, dec := SolarApparentEquatorial(e.jd)
		d := math.Abs(ra - e.ra)
		if d > 180 {
			d = 360 - d
		}
		if d > tol {
			t.Errorf("JD %v: apparent RA = %.6f, ephemeris %.6f (%.4f deg off)", e.jd, ra, e.ra, d)
		}
		if math.Abs(dec-e.dec) > tol {
			t.Errorf("JD %v: apparent Dec = %.6f, ephemeris %.6f", e.jd, dec, e.dec)
		}
		if r := SolarRadius(J2000Century(e.jd)); math.Abs(r-e.distance) > 1e-4 {
			t.Errorf("JD %v: radius = %.9f AU, ephemeris %.9f", e.jd, r, e.distance)
		}
	}
}
