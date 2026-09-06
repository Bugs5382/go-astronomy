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

// hoursToDeg converts hours, minutes, and seconds of right ascension to degrees.
func hoursToDeg(h, m, s float64) float64 { return hms(h, m, s) * 15 }

// TestEqToEcl anchors the equatorial-to-ecliptic transform against Meeus,
// Astronomical Algorithms, example 13.a: Pollux at right ascension
// 7h45m18.946s and declination +28d01'34.26", with the obliquity 23.4392911
// degrees, lies at ecliptic longitude 113.215630 and latitude +6.684170.
func TestEqToEcl(t *testing.T) {
	t.Parallel()
	lon, lat := EqToEcl(hoursToDeg(7, 45, 18.946), dms(28, 1, 34.26), 23.4392911)
	if got := pmod(lon, 360); math.Abs(got-113.215630) > 1e-5 {
		t.Errorf("ecliptic longitude = %.7f, want 113.215630", got)
	}
	if math.Abs(lat-6.684170) > 1e-5 {
		t.Errorf("ecliptic latitude = %.7f, want 6.684170", lat)
	}
}

// TestEclToEq inverts Meeus example 13.a.
func TestEclToEq(t *testing.T) {
	t.Parallel()
	ra, dec := EclToEq(113.215630, 6.684170, 23.4392911)
	wantRA := hoursToDeg(7, 45, 18.946)
	wantDec := dms(28, 1, 34.26)
	if got := pmod(ra, 360); math.Abs(got-wantRA) > 1e-4 {
		t.Errorf("RA = %.7f, want %.7f", got, wantRA)
	}
	if math.Abs(dec-wantDec) > 1e-4 {
		t.Errorf("Dec = %.7f, want %.7f", dec, wantDec)
	}
}

// TestEqToHz anchors the horizontal transform against Meeus example 13.b:
// Venus seen from Washington, DC on 1987 April 10 at 19:21 UT. The book works
// the example through a local hour angle of 64.352133 degrees and gives azimuth
// 68.0337 degrees measured westward from the south and altitude 15.1249
// degrees.
//
// The hour angle is supplied as the book states it, with zero right ascension
// and zero observer longitude, so this test checks formulae (13.5) and (13.6)
// alone and does not depend on the sidereal time.
func TestEqToHz(t *testing.T) {
	t.Parallel()
	az, alt := EqToHz(
		0,                  // right ascension folded into the hour angle
		-dms(6, 43, 11.61), // declination
		dms(38, 55, 17),    // observer latitude
		0,                  // observer longitude folded into the hour angle
		64.352133,          // local hour angle, degrees
	)
	if math.Abs(az-68.0337) > 1e-4 {
		t.Errorf("azimuth = %.6f, want 68.0337", az)
	}
	if math.Abs(alt-15.1249) > 1e-4 {
		t.Errorf("altitude = %.6f, want 15.1249", alt)
	}
}

// TestEqToHzFromSiderealTime runs Meeus example 13.b end to end, forming the
// hour angle from the apparent sidereal time at Greenwich rather than taking
// the book's rounded intermediate value. Apparent, not mean, is the sidereal
// time the example uses, since the coordinates given for Venus are apparent.
//
// The tolerance is a thousandth of a degree, about 4 arc seconds. The book
// prints its sidereal time to a ten-thousandth of a second of time, which is
// already half an arc second of hour angle, and rounds the hour angle it
// carries forward to six digits.
func TestEqToHzFromSiderealTime(t *testing.T) {
	t.Parallel()
	az, alt := EqToHz(
		hoursToDeg(23, 9, 16.641),
		-dms(6, 43, 11.61),
		dms(38, 55, 17),
		dms(77, 3, 56), // Washington is west of Greenwich
		ApparentSiderealTime(2446896.30625),
	)
	if math.Abs(az-68.0337) > 1e-3 {
		t.Errorf("azimuth = %.6f, want 68.0337", az)
	}
	if math.Abs(alt-15.1249) > 1e-3 {
		t.Errorf("altitude = %.6f, want 15.1249", alt)
	}
}

// TestHzToEq inverts Meeus example 13.b, recovering the declination and the
// hour angle the book started from.
func TestHzToEq(t *testing.T) {
	t.Parallel()
	ra, dec := HzToEq(68.0337, 15.1249, dms(38, 55, 17), 0, 0)
	// With zero sidereal time and zero longitude the returned right ascension
	// is the negated hour angle, reduced to one revolution.
	if got := pmod(-ra, 360); math.Abs(got-64.352133) > 1e-3 {
		t.Errorf("hour angle = %.6f, want 64.352133", got)
	}
	wantDec := -dms(6, 43, 11.61)
	if math.Abs(dec-wantDec) > 1e-3 {
		t.Errorf("Dec = %.6f, want %.6f", dec, wantDec)
	}
}

// TestEqHzRoundTrip walks positions through both horizontal transforms.
func TestEqHzRoundTrip(t *testing.T) {
	t.Parallel()
	const lat, lonWest, sidereal = 40.678, 73.944, 128.7378735
	cases := [][2]float64{{10, 20}, {200, -30}, {350, 5}, {120, 75}}
	for _, c := range cases {
		az, alt := EqToHz(c[0], c[1], lat, lonWest, sidereal)
		ra, dec := HzToEq(az, alt, lat, lonWest, sidereal)
		if math.Abs(pmod(ra, 360)-c[0]) > 1e-9 || math.Abs(dec-c[1]) > 1e-9 {
			t.Errorf("round trip (%v, %v) -> (%v, %v)", c[0], c[1], ra, dec)
		}
	}
}

// TestEqToHzMeridianTransit checks the geometry at the meridian, where the hour
// angle is zero: the body is due south of a northern observer whose latitude
// exceeds the declination, and its altitude is 90 minus their difference.
func TestEqToHzMeridianTransit(t *testing.T) {
	t.Parallel()
	const lat, dec = 40.678, 15.5
	// Sidereal time equal to the right ascension puts the body on the meridian.
	az, alt := EqToHz(120, dec, lat, 0, 120)
	if math.Abs(alt-(90-(lat-dec))) > 1e-9 {
		t.Errorf("transit altitude = %.9f, want %.9f", alt, 90-(lat-dec))
	}
	// Westward from the south, due south is zero.
	if math.Abs(az) > 1e-9 {
		t.Errorf("transit azimuth = %.9f, want 0 (due south)", az)
	}
}

// TestPrecessEqMeeusExample anchors the rigorous precession against Meeus
// example 21.b: theta Persei, at right ascension 2h44m11.986s and declination
// +49d13'42.48" in the J2000.0 frame with annual proper motion +0.03425 seconds
// of right ascension and -0.0895 arc seconds of declination, precessed to the
// equinox of JDE 2462088.69, gives 2h46m11.331s and +49d20'54.54".
func TestPrecessEqMeeusExample(t *testing.T) {
	t.Parallel()
	epochTo := JDEToJulianYear(2462088.69)
	years := epochTo - 2000
	// One second of right ascension is 1/240 degree.
	ra := hoursToDeg(2, 44, 11.986) + 0.03425*years/240
	dec := dms(49, 13, 42.48) - 0.0895*years/3600

	gotRA, gotDec := PrecessEq(ra, dec, 2000, epochTo)

	wantRA := hoursToDeg(2, 46, 11.331)
	wantDec := dms(49, 20, 54.54)
	const tol = 0.1 / 3600 // a tenth of an arc second, the book's precision
	if math.Abs(gotRA-wantRA) > tol {
		t.Errorf("RA = %.9f, want %.9f (%.4f arcsec off)", gotRA, wantRA, (gotRA-wantRA)*3600)
	}
	if math.Abs(gotDec-wantDec) > tol {
		t.Errorf("Dec = %.9f, want %.9f (%.4f arcsec off)", gotDec, wantDec, (gotDec-wantDec)*3600)
	}
}

// TestPrecessEqFromNonJ2000Epoch exercises the branch that rebuilds the
// precession angles for a starting epoch other than J2000.0, by precessing to
// B1950 and back and requiring the original position to return.
func TestPrecessEqFromNonJ2000Epoch(t *testing.T) {
	t.Parallel()
	cases := [][2]float64{
		{101.287, -16.716}, // Sirius
		{279.234, 38.784},  // Vega
		{37.95, 89.264},    // Polaris, close to the pole
		{0.5, 0},           // across the zero of right ascension
	}
	for _, c := range cases {
		ra1, dec1 := PrecessEq(c[0], c[1], 2000, 1950.0)
		ra2, dec2 := PrecessEq(ra1, dec1, 1950.0, 2000)
		if math.Abs(pmod(ra2, 360)-c[0]) > 1e-6 || math.Abs(dec2-c[1]) > 1e-6 {
			t.Errorf("round trip (%v, %v) -> (%v, %v)", c[0], c[1], ra2, dec2)
		}
	}
}

// TestPrecessEqNearPole checks the pole branch, which computes the declination
// from a hypotenuse rather than an arc sine because the arc sine loses
// precision where the cosine of the declination approaches zero. A position
// within a few arc seconds of the pole must still come back inside the
// physical range and must not become NaN.
func TestPrecessEqNearPole(t *testing.T) {
	t.Parallel()
	for _, dec := range []float64{89.9999, 90, -90, -89.9999} {
		ra, got := PrecessEq(45, dec, 2000, 2100)
		if math.IsNaN(ra) || math.IsNaN(got) {
			t.Fatalf("PrecessEq at dec %v returned NaN: (%v, %v)", dec, ra, got)
		}
		if got > 90 || got < -90 {
			t.Errorf("PrecessEq at dec %v gave %v, out of [-90, 90]", dec, got)
		}
		// A century of precession moves the pole by about 1.4 degrees, so the
		// declination must stay within that of the pole.
		if math.Abs(math.Abs(got)-90) > 2 {
			t.Errorf("PrecessEq at dec %v gave %v, too far from the pole", dec, got)
		}
	}
}

// TestPrecessEqIdentity checks that precessing across a zero interval is the
// identity transform.
func TestPrecessEqIdentity(t *testing.T) {
	t.Parallel()
	ra, dec := PrecessEq(101.287, -16.716, 2000, 2000)
	if math.Abs(pmod(ra, 360)-101.287) > 1e-12 || math.Abs(dec-(-16.716)) > 1e-12 {
		t.Errorf("identity precession changed the position: (%v, %v)", ra, dec)
	}
}
