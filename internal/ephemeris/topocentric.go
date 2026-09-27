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

// Topocentric place of a body.
//
// Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 11 ("The Earth's
// globe") gives the observer's geocentric position on the flattened Earth, and
// chapter 40 ("Correction for parallax") moves a geocentric place to the
// observer. Chapter 40 does it with closed-form corrections to right ascension
// and declination; here the same geometry is done with vectors, which also
// yields the observer-to-body distance the disc size needs (issue 44).

import "math"

// EarthEquatorialRadiusKm is the WGS84 equatorial radius of the Earth, in km.
const EarthEquatorialRadiusKm = 6378.137

// EarthFlattening is the WGS84 flattening of the Earth, (a - b) / a.
const EarthFlattening = 1 / 298.257223563

// ObserverParallaxConstants returns rho sin phi' and rho cos phi' for an
// observer at geodetic latitude latDeg and heightM metres above the ellipsoid,
// in units of the Earth's equatorial radius (Meeus 11.1 to 11.4). They are the
// observer's distance from the Earth's axis (rho cos phi') and from the
// equatorial plane (rho sin phi').
func ObserverParallaxConstants(latDeg, heightM float64) (rhoSinPhi, rhoCosPhi float64) {
	phi := radians(latDeg)
	ba := 1 - EarthFlattening
	u := math.Atan(ba * math.Tan(phi))
	h := heightM / (EarthEquatorialRadiusKm * 1000)
	rhoSinPhi = ba*math.Sin(u) + h*math.Sin(phi)
	rhoCosPhi = math.Cos(u) + h*math.Cos(phi)
	return rhoSinPhi, rhoCosPhi
}

// Topocentric moves a geocentric equatorial place (raDeg, decDeg, at distKm
// from the Earth's centre) to an observer at geodetic latitude latDeg and
// heightM metres, whose local sidereal time is lstDeg. It returns the
// topocentric right ascension in [0, 360), declination, and the distance from
// the observer to the body in km. Both vectors are expressed in the equatorial
// frame of date, so the difference is the rigorous parallax correction of
// Meeus chapter 40 with no small-angle approximation.
func Topocentric(raDeg, decDeg, distKm, latDeg, heightM, lstDeg float64) (raT, decT, distT float64) {
	ra, dec := radians(raDeg), radians(decDeg)
	bx := distKm * math.Cos(dec) * math.Cos(ra)
	by := distKm * math.Cos(dec) * math.Sin(ra)
	bz := distKm * math.Sin(dec)

	rs, rc := ObserverParallaxConstants(latDeg, heightM)
	theta := radians(lstDeg)
	ox := EarthEquatorialRadiusKm * rc * math.Cos(theta)
	oy := EarthEquatorialRadiusKm * rc * math.Sin(theta)
	oz := EarthEquatorialRadiusKm * rs

	x, y, z := bx-ox, by-oy, bz-oz
	distT = math.Sqrt(x*x + y*y + z*z)
	raT = pmod(degrees(math.Atan2(y, x)), 360)
	decT = degrees(math.Asin(z / distT))
	return raT, decT, distT
}
