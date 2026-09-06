package coordinates

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

// hms converts hours, minutes, seconds of right ascension to degrees.
func hms(h, m, s float64) float64 { return (h + m/60 + s/3600) * 15 }

// dms converts signed degrees, minutes, seconds to decimal degrees.
func dms(sign, d, m, s float64) float64 { return sign * (d + m/60 + s/3600) }

func TestEquatorialToEcliptic(t *testing.T) {
	t.Parallel()
	// Meeus example 13.a: Pollux.
	eq := Equatorial{RA: hms(7, 45, 18.946), Dec: dms(1, 28, 1, 34.26)}
	const obliquity = 23.4392911
	ecl := EquatorialToEcliptic(eq, obliquity)

	if math.Abs(ecl.Lon-113.21563) > 1e-4 {
		t.Errorf("ecliptic longitude = %.6f, want 113.21563", ecl.Lon)
	}
	if math.Abs(ecl.Lat-6.68417) > 1e-4 {
		t.Errorf("ecliptic latitude = %.6f, want 6.68417", ecl.Lat)
	}
}

func TestEclipticToEquatorial(t *testing.T) {
	t.Parallel()
	// Inverse of Meeus example 13.a.
	ecl := Ecliptic{Lon: 113.21563, Lat: 6.68417}
	const obliquity = 23.4392911
	eq := EclipticToEquatorial(ecl, obliquity)

	if math.Abs(eq.RA-hms(7, 45, 18.946)) > 1e-3 {
		t.Errorf("RA = %.6f, want %.6f", eq.RA, hms(7, 45, 18.946))
	}
	if math.Abs(eq.Dec-dms(1, 28, 1, 34.26)) > 1e-3 {
		t.Errorf("Dec = %.6f, want %.6f", eq.Dec, dms(1, 28, 1, 34.26))
	}
}

func TestEclipticEquatorialRoundTrip(t *testing.T) {
	t.Parallel()
	const obliquity = 23.4392911
	cases := []Equatorial{
		{RA: 0, Dec: 0},
		{RA: 116.328942, Dec: 28.026183},
		{RA: 250, Dec: -45},
		{RA: 350, Dec: 80},
	}
	for _, eq := range cases {
		got := EclipticToEquatorial(EquatorialToEcliptic(eq, obliquity), obliquity)
		if math.Abs(got.RA-eq.RA) > 1e-6 || math.Abs(got.Dec-eq.Dec) > 1e-6 {
			t.Errorf("round trip %+v -> %+v", eq, got)
		}
	}
}

func TestEquatorialToHorizontal(t *testing.T) {
	t.Parallel()
	// Meeus example 13.b: Venus from Washington, DC on 1987 April 10 19:21 UT.
	// The example's own sidereal time at Greenwich is 8h34m57.0896s (example
	// 12.b), used here as given so the transform is checked against the book
	// rather than against another computation in this module.
	gst := hms(8, 34, 57.0896)

	eq := Equatorial{RA: hms(23, 9, 16.641), Dec: dms(-1, 6, 43, 11.61)}
	lat := dms(1, 38, 55, 17)
	lonEast := -dms(1, 77, 3, 56) // Washington is west of Greenwich

	hz := EquatorialToHorizontal(eq, lat, lonEast, gst)

	// Meeus reports azimuth 68.034 deg measured westward from south, which is
	// 248.034 deg measured clockwise from north, and altitude 15.125 deg.
	if math.Abs(hz.Azimuth-248.034) > 5e-3 {
		t.Errorf("azimuth = %.4f, want 248.034", hz.Azimuth)
	}
	if math.Abs(hz.Altitude-15.125) > 5e-3 {
		t.Errorf("altitude = %.4f, want 15.125", hz.Altitude)
	}
	if hz.Azimuth < 0 || hz.Azimuth >= 360 {
		t.Errorf("azimuth = %v out of [0,360)", hz.Azimuth)
	}
}

func TestHorizontalToEquatorial(t *testing.T) {
	t.Parallel()
	// Inverse of Meeus example 13.b, with the book's sidereal time (12.b).
	gst := hms(8, 34, 57.0896)

	hz := Horizontal{Azimuth: 248.0337, Altitude: 15.1249}
	lat := dms(1, 38, 55, 17)
	lonEast := -dms(1, 77, 3, 56)

	eq := HorizontalToEquatorial(hz, lat, lonEast, gst)

	if math.Abs(eq.RA-hms(23, 9, 16.641)) > 1e-2 {
		t.Errorf("RA = %.6f, want %.6f", eq.RA, hms(23, 9, 16.641))
	}
	if math.Abs(eq.Dec-dms(-1, 6, 43, 11.61)) > 1e-2 {
		t.Errorf("Dec = %.6f, want %.6f", eq.Dec, dms(-1, 6, 43, 11.61))
	}
}

func TestEquatorialHorizontalRoundTrip(t *testing.T) {
	t.Parallel()
	const gst = 128.7378735
	lat, lonEast := 40.678, -73.944 // Brooklyn, NY
	cases := []Equatorial{
		{RA: 10, Dec: 20},
		{RA: 200, Dec: -30},
		{RA: 350, Dec: 5},
	}
	for _, eq := range cases {
		hz := EquatorialToHorizontal(eq, lat, lonEast, gst)
		if hz.Azimuth < 0 || hz.Azimuth >= 360 {
			t.Errorf("azimuth = %v out of [0,360)", hz.Azimuth)
		}
		got := HorizontalToEquatorial(hz, lat, lonEast, gst)
		if math.Abs(got.RA-eq.RA) > 1e-6 || math.Abs(got.Dec-eq.Dec) > 1e-6 {
			t.Errorf("round trip %+v -> %+v", eq, got)
		}
	}
}

func TestPrecessEquatorialRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []Equatorial{
		{RA: 101.287, Dec: -16.716}, // Sirius
		{RA: 37.95, Dec: 89.264},    // Polaris
		{RA: 279.234, Dec: 38.784},  // Vega
		{RA: 0.5, Dec: 0},
	}
	for _, eq := range cases {
		back := PrecessEquatorial(PrecessEquatorial(eq, 2000, 1875.0287), 1875.0287, 2000)
		if math.Abs(back.RA-eq.RA) > 1e-6 || math.Abs(back.Dec-eq.Dec) > 1e-6 {
			t.Errorf("precession round trip %+v -> %+v", eq, back)
		}
		if back.RA < 0 || back.RA >= 360 {
			t.Errorf("RA %v out of [0,360)", back.RA)
		}
	}
}

// TestPrecessEquatorialMovesPosition confirms precession actually shifts a
// mid-declination star: over a century-plus the equinox drift is well over a
// degree in right ascension.
func TestPrecessEquatorialMovesPosition(t *testing.T) {
	t.Parallel()
	eq := Equatorial{RA: 101.287, Dec: -16.716} // Sirius
	got := PrecessEquatorial(eq, 2000, 1875.0287)
	if math.Abs(got.RA-eq.RA) < 1.0 {
		t.Errorf("precession barely moved RA: %v -> %v", eq.RA, got.RA)
	}
}

// TestPrecessEquatorialMeeusExample anchors the rigorous precession against
// Meeus, Astronomical Algorithms, example 21.b: theta Persei from the J2000.0
// equinox to the equinox of JDE 2462088.69 (Julian year 2028.867050), for which
// the book gives alpha = 2h46m11.331s and delta = +49d20'54.54".
//
// The book applies the star's annual proper motion before precessing.
// PrecessEquatorial deliberately models the precession of the equinoxes only,
// so the proper motion is applied here, in the test, exactly as the example
// does it: the catalog position advances by the annual rates times the interval
// in Julian years, then the result is precessed.
func TestPrecessEquatorialMeeusExample(t *testing.T) {
	t.Parallel()
	const (
		epochTo = 2028.867050 // Julian year of JDE 2462088.69
		years   = epochTo - 2000
		// Annual proper motion: +0.03425 seconds of right ascension and
		// -0.0895 arc seconds of declination per Julian year.
		pmRASec  = 0.03425
		pmDecSec = -0.0895
	)
	// One second of right ascension is fifteen arc seconds, so 1/240 degree.
	eq := Equatorial{
		RA:  hms(2, 44, 11.986) + pmRASec*years/240,
		Dec: dms(1, 49, 13, 42.48) + pmDecSec*years/3600,
	}

	got := PrecessEquatorial(eq, 2000, epochTo)

	wantRA := hms(2, 46, 11.331)
	wantDec := dms(1, 49, 20, 54.54)
	// A tenth of an arc second, the precision the book quotes.
	const tol = 0.1 / 3600
	if math.Abs(got.RA-wantRA) > tol {
		t.Errorf("RA = %.9f deg, want %.9f deg (%.4f arcsec off)",
			got.RA, wantRA, (got.RA-wantRA)*3600)
	}
	if math.Abs(got.Dec-wantDec) > tol {
		t.Errorf("Dec = %.9f deg, want %.9f deg (%.4f arcsec off)",
			got.Dec, wantDec, (got.Dec-wantDec)*3600)
	}
}
