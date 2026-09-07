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

// example47a is the instant of Meeus, Astronomical Algorithms, example 47.a:
// 1992 April 12.0 TD.
const example47a = 2448724.5

// TestMoonPosition anchors the truncated lunar series against Meeus example
// 47.a, which gives geocentric ecliptic longitude 133.162655 degrees, latitude
// -3.229126 degrees, and an Earth-Moon distance of 368409.7 kilometres.
//
// The longitude and latitude are referred to the mean equinox of date and do
// not include nutation, which is what the book's example computes.
func TestMoonPosition(t *testing.T) {
	t.Parallel()
	lon, lat, dist := MoonPosition(example47a)
	if math.Abs(lon-133.162655) > 1e-6 {
		t.Errorf("longitude = %.7f, want 133.162655", lon)
	}
	if math.Abs(lat-(-3.229126)) > 1e-6 {
		t.Errorf("latitude = %.7f, want -3.229126", lat)
	}
	if math.Abs(dist-368409.7) > 0.05 {
		t.Errorf("distance = %.2f km, want 368409.7", dist)
	}
	if lon < 0 || lon >= 360 {
		t.Errorf("longitude = %v out of [0,360)", lon)
	}
}

// TestMoonParallax anchors the equatorial horizontal parallax against Meeus
// example 47.a, which gives 0.991990 degrees at a distance of 368409.7
// kilometres.
func TestMoonParallax(t *testing.T) {
	t.Parallel()
	if got := MoonParallax(368409.7); math.Abs(got-0.991990) > 1e-6 {
		t.Errorf("MoonParallax = %.7f, want 0.991990", got)
	}
	// Parallax falls as the Moon recedes.
	if MoonParallax(400000) >= MoonParallax(360000) {
		t.Error("parallax did not decrease with distance")
	}
}

// TestMoonPositionRanges walks four years at six-hour steps and checks the
// series stays inside the Moon's physical bounds: the geocentric latitude never
// leaves the roughly 5.3 degree band set by the inclination of the lunar orbit
// plus its perturbations, and the distance stays between perigee and apogee.
func TestMoonPositionRanges(t *testing.T) {
	t.Parallel()
	var minDist, maxDist = math.Inf(1), math.Inf(-1)
	var maxLat float64
	for i := 0; i < 4*365*4; i++ {
		jde := J2000 + float64(i)*0.25
		lon, lat, dist := MoonPosition(jde)
		if lon < 0 || lon >= 360 {
			t.Fatalf("longitude %v out of [0,360) at JDE %v", lon, jde)
		}
		if math.Abs(lat) > 5.5 {
			t.Fatalf("latitude %v exceeds the lunar orbit's inclination at JDE %v", lat, jde)
		}
		maxLat = math.Max(maxLat, math.Abs(lat))
		minDist = math.Min(minDist, dist)
		maxDist = math.Max(maxDist, dist)
	}
	// The Moon reaches at least 5 degrees of ecliptic latitude within four
	// years, which confirms the latitude series is not stuck near zero.
	if maxLat < 5 {
		t.Errorf("maximum latitude over four years was only %.3f degrees", maxLat)
	}
	if minDist < 356000 || minDist > 358000 {
		t.Errorf("minimum distance %.1f km is not near perigee", minDist)
	}
	if maxDist < 406000 || maxDist > 407000 {
		t.Errorf("maximum distance %.1f km is not near apogee", maxDist)
	}
}

// TestMoonPositionSynodicPeriod checks the Moon completes a revolution in
// longitude in one sidereal month, 27.32 days, by counting the wraps over a
// long span.
func TestMoonPositionSynodicPeriod(t *testing.T) {
	t.Parallel()
	const span = 2732.0 // days, about a hundred sidereal months
	var wraps int
	prev, _, _ := MoonPosition(J2000)
	for i := 1; i <= int(span*24); i++ {
		lon, _, _ := MoonPosition(J2000 + float64(i)/24)
		if lon < prev {
			wraps++
		}
		prev = lon
	}
	if wraps != 100 {
		t.Errorf("counted %d revolutions over %.0f days, want 100", wraps, span)
	}
}

// ephemerisMoon is the Moon's apparent geocentric ecliptic longitude and
// latitude of date, in degrees, and its geocentric distance, from the JPL
// Horizons system (target 301, center 500@399) evaluated in Terrestrial Time so
// that the omission of Delta-T in this library does not enter the comparison.
// Horizons reports the distance in astronomical units; the kilometre values are
// that figure times 149597870.700, the IAU 2012 definition of the unit.
var ephemerisMoon = []struct {
	jde, lon, lat, distanceKm float64
}{
	{2448724.5, 133.1667103, -3.2292008, 368439.4},
	{2461059.5, 300.8776713, -3.2189482, 394864.0},
	{2461212.5, 168.6863793, -1.4713155, 383138.4},
}

// TestMoonPositionAgainstEphemeris measures the truncated lunar series against
// a modern numerical ephemeris.
//
// Meeus gives the accuracy of the abridged chapter 47 series as about 10 arc
// seconds in longitude and 4 arc seconds in latitude. The comparison must also
// allow for nutation in longitude, up to about 17 arc seconds, because the
// series returns the position referred to the mean equinox of date while the
// ephemeris reports it referred to the true equinox; nutation is added here so
// the two are on the same footing.
func TestMoonPositionAgainstEphemeris(t *testing.T) {
	t.Parallel()
	for _, e := range ephemerisMoon {
		lon, lat, dist := MoonPosition(e.jde)
		dpsi, _ := Nutation(e.jde)
		apparentLon := pmod(lon+dpsi, 360)
		if d := math.Abs(apparentLon - e.lon); d > 15.0/3600 {
			t.Errorf("JDE %v: apparent longitude = %.7f, ephemeris %.7f (%.2f arcsec off)",
				e.jde, apparentLon, e.lon, d*3600)
		}
		if d := math.Abs(lat - e.lat); d > 6.0/3600 {
			t.Errorf("JDE %v: latitude = %.7f, ephemeris %.7f (%.2f arcsec off)",
				e.jde, lat, e.lat, d*3600)
		}
		// The distance series is truncated too; 50 km on 368000 is about a
		// hundredth of a percent, well below the apparent-diameter precision
		// this library needs.
		if d := math.Abs(dist - e.distanceKm); d > 50 {
			t.Errorf("JDE %v: distance = %.1f km, ephemeris %.1f (%.1f km off)",
				e.jde, dist, e.distanceKm, d)
		}
	}
}
