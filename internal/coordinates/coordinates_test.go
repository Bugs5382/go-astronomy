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
	"time"

	meeusjulian "github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/sidereal"
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
	jd := meeusjulian.TimeToJD(time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC))
	gst := sidereal.Apparent(jd).Hour() * 15

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
	// Inverse of Meeus example 13.b.
	jd := meeusjulian.TimeToJD(time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC))
	gst := sidereal.Apparent(jd).Hour() * 15

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
