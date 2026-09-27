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

// TestObserverParallaxConstants anchors the geocentric observer terms against
// Meeus, Astronomical Algorithms, example 11.a: Palomar Observatory at
// latitude +33 21 22 and 1706 m gives rho sin phi' = 0.546861 and
// rho cos phi' = 0.836339. The book uses the IAU 1976 ellipsoid; the WGS84
// ellipsoid used here moves both values by under 1e-6.
func TestObserverParallaxConstants(t *testing.T) {
	t.Parallel()
	rs, rc := ObserverParallaxConstants(dms(33, 21, 22), 1706)
	if got, want := rs, 0.546861; math.Abs(got-want) > 2e-6 {
		t.Errorf("rho sin phi' = %.6f, want %.6f", got, want)
	}
	if got, want := rc, 0.836339; math.Abs(got-want) > 2e-6 {
		t.Errorf("rho cos phi' = %.6f, want %.6f", got, want)
	}
}

// TestTopocentricMeeusExample anchors the rigorous parallax correction against
// Meeus example 40.a: Mars on 2003 August 28 at 3h17m UT from Palomar, at a
// geocentric right ascension of 22h38m07.25s, declination -15 46 15.9, and a
// distance of 0.37276 AU, with Greenwich apparent sidereal time 1h40m45s. The
// book gives a topocentric right ascension of 22h38m08.54s and declination of
// -15 46 30.0.
func TestTopocentricMeeusExample(t *testing.T) {
	t.Parallel()
	const auKm = 149597870.7
	ra := hms(22, 38, 7.25) * 15
	dec := -dms(15, 46, 15.9)
	lst := hms(1, 40, 45)*15 - hms(7, 47, 27)*15
	dist := 0.37276 * auKm
	gotRA, gotDec, gotDist := Topocentric(ra, dec, dist, dms(33, 21, 22), 1706, lst)
	if want := hms(22, 38, 8.54) * 15; math.Abs(gotRA-want)*3600 > 0.15 {
		t.Errorf("topocentric RA = %.6f deg, want %.6f (off %.3f arcsec)", gotRA, want, (gotRA-want)*3600)
	}
	if want := -dms(15, 46, 30.0); math.Abs(gotDec-want)*3600 > 0.15 {
		t.Errorf("topocentric Dec = %.6f deg, want %.6f (off %.3f arcsec)", gotDec, want, (gotDec-want)*3600)
	}
	if gotDist >= dist || gotDist < dist-EarthEquatorialRadiusKm {
		t.Errorf("topocentric distance = %.1f km, want within an Earth radius below %.1f", gotDist, dist)
	}
}

// TestTopocentricZenith checks the geometry directly: a body straight overhead
// of an observer on the equator at sea level is nearer by exactly the
// equatorial radius, and its direction does not move.
func TestTopocentricZenith(t *testing.T) {
	t.Parallel()
	ra, dec, dist := Topocentric(120, 0, 384400, 0, 0, 120)
	if math.Abs(ra-120) > 1e-9 || math.Abs(dec) > 1e-9 {
		t.Errorf("zenith body moved to RA %.9f Dec %.9f", ra, dec)
	}
	if want := 384400 - EarthEquatorialRadiusKm; math.Abs(dist-want) > 1e-6 {
		t.Errorf("zenith distance = %.6f km, want %.6f", dist, want)
	}
}

// TestTopocentricPoleDistance checks the flattening: an observer at the north
// pole sits b = a(1 - f) from the centre, so a body over the pole is nearer by
// the polar radius rather than the equatorial one.
func TestTopocentricPoleDistance(t *testing.T) {
	t.Parallel()
	_, dec, dist := Topocentric(0, 90, 384400, 90, 0, 0)
	if math.Abs(dec-90) > 1e-9 {
		t.Errorf("polar body moved to Dec %.9f", dec)
	}
	polar := EarthEquatorialRadiusKm * (1 - EarthFlattening)
	if want := 384400 - polar; math.Abs(dist-want) > 1e-6 {
		t.Errorf("polar distance = %.6f km, want %.6f", dist, want)
	}
}
