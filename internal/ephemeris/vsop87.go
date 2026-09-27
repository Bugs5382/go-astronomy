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

// vsop87Term is one periodic term A cos(B + C tau), with A in radians or
// astronomical units, B in radians, and C in radians per Julian millennium.
type vsop87Term struct{ A, B, C float64 }

// vsop87Body holds the three VSOP87D variables of one body, each a list of
// series indexed by the power of tau that multiplies it.
type vsop87Body struct {
	L, B, R [][]vsop87Term
}

// vsop87Sum evaluates sum over alpha of tau^alpha times sum A cos(B + C tau).
func vsop87Sum(series [][]vsop87Term, tau float64) float64 {
	var total float64
	for alpha := len(series) - 1; alpha >= 0; alpha-- {
		var s float64
		for _, t := range series[alpha] {
			s += t.A * math.Cos(t.B+t.C*tau)
		}
		total = total*tau + s
	}
	return total
}

// J2000Millennium returns the Julian millennia from J2000.0 to the given
// Julian ephemeris day, the time argument tau of VSOP87.
func J2000Millennium(jde float64) float64 {
	return (jde - J2000) / (10 * JulianCentury)
}

// EarthHeliocentric returns the Earth's heliocentric ecliptic longitude, in
// degrees in [0, 360), latitude in degrees, and radius vector in astronomical
// units, at the given Julian ephemeris day. The angles are referred to the
// dynamical ecliptic and equinox of date (VSOP87D).
func EarthHeliocentric(jde float64) (lonDeg, latDeg, radiusAU float64) {
	tau := J2000Millennium(jde)
	l := vsop87Sum(earthVSOP87D.L, tau)
	b := vsop87Sum(earthVSOP87D.B, tau)
	r := vsop87Sum(earthVSOP87D.R, tau)
	return pmod(degrees(l), 360), degrees(b), r
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
	lon, lat, r := SunGeometric(jde)
	dpsi, _ := Nutation(jde)
	return pmod(lon+dpsi-20.4898/arcsecPerDeg/r, 360), lat, r
}

// SunApparent returns the Sun's apparent geocentric right ascension, in
// degrees in [0, 360), declination in degrees, and distance in km at the given
// Julian ephemeris day, referred to the true equator and equinox of date
// (issue 45).
func SunApparent(jde float64) (raDeg, decDeg, distKm float64) {
	lon, lat, r := SunApparentEcliptic(jde)
	raDeg, decDeg = EclToEq(lon, lat, TrueObliquity(jde))
	return raDeg, decDeg, r * KmPerAU
}
