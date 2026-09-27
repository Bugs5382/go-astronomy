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

// The Earth's heliocentric position from VSOP87, and the Sun's geocentric
// position that follows from it.
//
// The theory is Bretagnon and Francou, "Planetary theories in rectangular and
// spherical variables. VSOP87 solutions", Astronomy and Astrophysics 202, 309
// (1988), in its version D: heliocentric ecliptic longitude, latitude, and
// radius referred to the dynamical ecliptic and equinox of date. The table is
// generated from the IMCCE files by internal/cmd/genvsop and truncated by
// amplitude; see the header of vsop87_earth.go. The reduction to the Sun's
// apparent place follows Jean Meeus, Astronomical Algorithms, 2nd ed.,
// chapter 25, "Solar Coordinates", the higher-accuracy method: formula (25.9)
// for the conversion to the FK5 system, then nutation and aberration.

//go:generate go run ../cmd/genvsop -body earth -var earthVSOP87D -out vsop87_earth.go

import "math"

// EarthHeliocentric returns the Earth's heliocentric ecliptic longitude, in
// degrees in [0, 360), latitude in degrees, and radius vector in astronomical
// units, at the given Julian ephemeris day. The angles are referred to the
// dynamical ecliptic and equinox of date (VSOP87D).
func EarthHeliocentric(jde float64) (lonDeg, latDeg, radiusAU float64) {
	return earthVSOP87D.Heliocentric(jde)
}

// SunGeometric returns the Sun's geometric geocentric ecliptic longitude, in
// degrees in [0, 360), latitude in degrees, and distance in astronomical
// units, referred to the FK5 system and the mean equinox of date, at the
// given Julian ephemeris day.
//
// The Sun is seen from the Earth in the opposite direction to the Earth seen
// from the Sun. The conversion from the dynamical frame of VSOP87 to FK5 is
// formula (25.9).
func SunGeometric(jde float64) (lonDeg, latDeg, distanceAU float64) {
	l, b, r := EarthHeliocentric(jde)
	lon := l + 180
	lat := -b
	t := J2000Century(jde)
	lp := radians(lon - 1.397*t - 0.00031*t*t)
	lon += -0.09033 / arcsecPerDeg
	lat += 0.03916 / arcsecPerDeg * (math.Cos(lp) - math.Sin(lp))
	return pmod(lon, 360), lat, r
}

// SunApparentEcliptic returns the Sun's apparent geocentric ecliptic
// longitude, in degrees in [0, 360), and latitude in degrees, referred to the
// true equinox of date, and its distance in astronomical units, at the given
// Julian ephemeris day. It adds the nutation in longitude and the aberration
// for the actual distance, 20.4898 arc seconds divided by the distance in
// astronomical units (Meeus 25.10).
func SunApparentEcliptic(jde float64) (lonDeg, latDeg, distanceAU float64) {
	dpsi, _ := Nutation(jde)
	return sunApparentEcliptic(jde, dpsi)
}

// sunApparentEcliptic is SunApparentEcliptic with the nutation in longitude
// already known.
func sunApparentEcliptic(jde, dpsi float64) (lonDeg, latDeg, distanceAU float64) {
	lon, lat, r := SunGeometric(jde)
	return pmod(lon+dpsi-20.4898/arcsecPerDeg/r, 360), lat, r
}

// SunApparent returns the Sun's apparent geocentric right ascension, in
// degrees in [0, 360), declination in degrees, and distance in km at the given
// Julian ephemeris day, referred to the true equator and equinox of date
// (issue 45).
func SunApparent(jde float64) (raDeg, decDeg, distKm float64) {
	return SunApparentNutated(jde, NutationAt(jde))
}

// SunApparentNutated is SunApparent with the nutation at jde already
// evaluated, for a caller that needs it again (the apparent sidereal time).
// It gives exactly SunApparent's result.
func SunApparentNutated(jde float64, n Nutated) (raDeg, decDeg, distKm float64) {
	lon, lat, r := sunApparentEcliptic(jde, n.DPsi)
	raDeg, decDeg = EclToEq(lon, lat, n.TrueObliquity())
	return raDeg, decDeg, r * KmPerAU
}

// FK5Correction returns the corrections, in degrees, that take a geocentric
// ecliptic longitude and latitude computed from VSOP87 (the dynamical frame)
// to the FK5 system, at the given Julian ephemeris day (Meeus 32.3). SunGeometric
// applies the same correction with the latitude taken as zero.
func FK5Correction(lonDeg, latDeg, jde float64) (dLonDeg, dLatDeg float64) {
	t := J2000Century(jde)
	lp := radians(lonDeg - 1.397*t - 0.00031*t*t)
	c, s := math.Cos(lp), math.Sin(lp)
	dLonDeg = (-0.09033 + 0.03916*(c+s)*math.Tan(radians(latDeg))) / arcsecPerDeg
	dLatDeg = 0.03916 * (c - s) / arcsecPerDeg
	return dLonDeg, dLatDeg
}
