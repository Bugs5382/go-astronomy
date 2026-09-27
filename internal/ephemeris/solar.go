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

// Solar coordinates.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 25,
// "Solar Coordinates": formulae (25.2) through (25.8) and the low-accuracy
// aberration correction (25.10). The Go form derives from soniakeys/meeus
// (solar), MIT licensed.
//
// This is the book's lower-accuracy method, good to about 0.01 degrees in
// longitude. The Sun's position now comes from VSOP87 (vsop87.go); these
// functions stay, anchored to the book's worked examples, as the independent
// cross-check the VSOP87 tests measure against.

import "math"

// SolarMeanLongitude returns the geometric mean longitude of the Sun, in
// degrees, referred to the mean equinox of date, at T Julian centuries from
// J2000.0 (Meeus 25.2). The result is not reduced to one revolution.
func SolarMeanLongitude(t float64) float64 {
	return Horner(t, 280.46646, 36000.76983, 0.0003032)
}

// SolarMeanAnomaly returns the mean anomaly of the Earth, in degrees, at T
// Julian centuries from J2000.0 (Meeus 25.3). The result is not reduced to one
// revolution.
func SolarMeanAnomaly(t float64) float64 {
	return Horner(t, 357.52911, 35999.05029, -0.0001537)
}

// SolarEquationOfCenter returns the equation of the center of the Sun, in
// degrees, at T Julian centuries from J2000.0: the difference between the true
// and the mean position, as a series in the mean anomaly (Meeus, p. 164).
func SolarEquationOfCenter(t float64) float64 {
	m := radians(SolarMeanAnomaly(t))
	return Horner(t, 1.914602, -0.004817, -0.000014)*math.Sin(m) +
		(0.019993-0.000101*t)*math.Sin(2*m) +
		0.000289*math.Sin(3*m)
}

// SolarTrueLongitude returns the Sun's true geometric longitude and the
// Earth's true anomaly, both in degrees in [0, 360), referred to the mean
// equinox of date, at T Julian centuries from J2000.0. Each is the
// corresponding mean value plus the equation of the center.
func SolarTrueLongitude(t float64) (lonDeg, anomalyDeg float64) {
	c := SolarEquationOfCenter(t)
	return pmod(SolarMeanLongitude(t)+c, 360), pmod(SolarMeanAnomaly(t)+c, 360)
}

// SolarEccentricity returns the eccentricity of the Earth's orbit at T Julian
// centuries from J2000.0 (Meeus 25.4).
func SolarEccentricity(t float64) float64 {
	return Horner(t, 0.016708634, -0.000042037, -0.0000001267)
}

// SolarRadius returns the distance from the Earth to the Sun, in astronomical
// units, at T Julian centuries from J2000.0 (Meeus 25.5). It is the radius
// vector of the elliptical orbit evaluated at the true anomaly.
func SolarRadius(t float64) float64 {
	_, anomaly := SolarTrueLongitude(t)
	e := SolarEccentricity(t)
	return 1.000001018 * (1 - e*e) / (1 + e*math.Cos(radians(anomaly)))
}

// solarNode returns the longitude of the ascending node of the Moon's mean
// orbit, in degrees, in the abridged form the solar corrections use (Meeus,
// p. 164). It is not the full series of chapter 47.
func solarNode(t float64) float64 {
	return 125.04 - 1934.136*t
}

// SolarApparentLongitude returns the Sun's apparent longitude, in degrees in
// [0, 360), referred to the true equinox of date, at T Julian centuries from
// J2000.0 (Meeus, p. 164).
//
// The constant -0.00569 degrees is the aberration of light at the Earth's mean
// distance, formula (25.10) evaluated at one astronomical unit, and the term in
// the lunar node is the nutation in longitude in its abridged form.
func SolarApparentLongitude(t float64) float64 {
	lon, _ := SolarTrueLongitude(t)
	return pmod(lon-0.00569-0.00478*math.Sin(radians(solarNode(t))), 360)
}

// SolarApparentEquatorial returns the Sun's apparent right ascension, in
// degrees in [0, 360), and declination, in degrees, at the given Julian
// ephemeris day (Meeus 25.8).
//
// The obliquity carries the correction of formula (25.8), which stands in for
// the nutation in obliquity, so the result is referred to the true equator and
// equinox of date and is consistent with apparent sidereal time.
func SolarApparentEquatorial(jde float64) (raDeg, decDeg float64) {
	t := J2000Century(jde)
	lon := radians(SolarApparentLongitude(t))
	eps := radians(MeanObliquity(jde) + 0.00256*math.Cos(radians(solarNode(t))))
	sinLon, cosLon := math.Sincos(lon)
	sinEps, cosEps := math.Sincos(eps)
	ra := math.Atan2(cosEps*sinLon, cosLon)
	dec := math.Asin(sinEps * sinLon)
	return pmod(degrees(ra), 360), degrees(dec)
}
