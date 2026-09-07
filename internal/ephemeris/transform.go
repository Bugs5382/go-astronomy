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

// Coordinate transforms and the rigorous precession of equatorial coordinates.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed.: chapter 13,
// "Transformation of Coordinates", formulae (13.1) through (13.6); and
// chapter 21, "Precession", formulae (21.2), (21.3), and (21.4). The Go form
// derives from soniakeys/meeus (coord, precess), MIT licensed.
//
// Angles are degrees in and degrees out. The horizontal functions keep the
// book's conventions -- observer longitude positive to the west, azimuth
// measured westward from the south -- so that they can be checked directly
// against the worked examples. Rotating to the conventions this library
// publishes is the caller's job; internal/coordinates does it.

import "math"

// EqToEcl converts an equatorial position to ecliptic longitude and latitude,
// in degrees, given the obliquity of the ecliptic in degrees (Meeus 13.1,
// 13.2). Longitude is not reduced to one revolution.
func EqToEcl(raDeg, decDeg, obliquityDeg float64) (lonDeg, latDeg float64) {
	sinEps, cosEps := math.Sincos(radians(obliquityDeg))
	sinRA, cosRA := math.Sincos(radians(raDeg))
	sinDec, cosDec := math.Sincos(radians(decDeg))
	lon := math.Atan2(sinRA*cosEps+(sinDec/cosDec)*sinEps, cosRA)
	lat := math.Asin(sinDec*cosEps - cosDec*sinEps*sinRA)
	return degrees(lon), degrees(lat)
}

// EclToEq converts an ecliptic position to equatorial right ascension and
// declination, in degrees, given the obliquity of the ecliptic in degrees
// (Meeus 13.3, 13.4). Right ascension is reduced to [0, 360).
func EclToEq(lonDeg, latDeg, obliquityDeg float64) (raDeg, decDeg float64) {
	sinEps, cosEps := math.Sincos(radians(obliquityDeg))
	sinLon, cosLon := math.Sincos(radians(lonDeg))
	sinLat, cosLat := math.Sincos(radians(latDeg))
	ra := math.Atan2(sinLon*cosEps-(sinLat/cosLat)*sinEps, cosLon)
	dec := math.Asin(sinLat*cosEps + cosLat*sinEps*sinLon)
	return pmod(degrees(ra), 360), degrees(dec)
}

// EqToHz converts an equatorial position to horizontal coordinates for an
// observer at the given latitude and west-positive longitude, in degrees, with
// siderealDeg the sidereal time at Greenwich in degrees (Meeus 13.5, 13.6).
//
// Azimuth is measured westward from the south, the book's convention. The
// sidereal time must match the coordinates: mean with mean, apparent with
// apparent.
func EqToHz(raDeg, decDeg, latDeg, lonWestDeg, siderealDeg float64) (azDeg, altDeg float64) {
	h := radians(siderealDeg - lonWestDeg - raDeg)
	sinH, cosH := math.Sincos(h)
	sinLat, cosLat := math.Sincos(radians(latDeg))
	sinDec, cosDec := math.Sincos(radians(decDeg))
	az := math.Atan2(sinH, cosH*sinLat-(sinDec/cosDec)*cosLat)
	alt := math.Asin(sinLat*sinDec + cosLat*cosDec*cosH)
	return degrees(az), degrees(alt)
}

// HzToEq converts horizontal coordinates back to equatorial right ascension and
// declination, in degrees. It is the inverse of EqToHz and takes the same
// conventions: west-positive observer longitude and azimuth westward from the
// south. Right ascension is reduced to [0, 360).
func HzToEq(azDeg, altDeg, latDeg, lonWestDeg, siderealDeg float64) (raDeg, decDeg float64) {
	sinAz, cosAz := math.Sincos(radians(azDeg))
	sinAlt, cosAlt := math.Sincos(radians(altDeg))
	sinLat, cosLat := math.Sincos(radians(latDeg))
	h := math.Atan2(sinAz, cosAz*sinLat+(sinAlt/cosAlt)*cosLat)
	ra := radians(siderealDeg-lonWestDeg) - h
	dec := math.Asin(sinLat*sinAlt - cosLat*cosAlt*cosAz)
	return pmod(degrees(ra), 360), degrees(dec)
}

// arcsec converts arc seconds to radians, the unit the precession series is
// printed in.
func arcsec(s float64) float64 { return radians(s / arcsecPerDeg) }

// Precession coefficients from Meeus (21.2), the general case, and (21.3), the
// case of a starting epoch of J2000.0. Each triple is a polynomial in T, the
// centuries from J2000.0 to the starting epoch.
var (
	zetaT  = [3]float64{2306.2181, 1.39656, -0.000139}
	zT     = [3]float64{2306.2181, 1.39656, -0.000139} // same first-order series as zeta
	thetaT = [3]float64{2004.3109, -0.8533, -0.000217}
)

// cosSmallAngle is the cosine of ten arc minutes, the threshold Meeus
// recommends (chapter 17, p. 109) for switching between an arc sine and a
// Pythagorean form near a singularity.
var cosSmallAngle = math.Cos(radians(10.0 / 60))

// PrecessEq precesses an equatorial position from the equinox of epochFrom to
// the equinox of epochTo, both Julian years, and returns right ascension in
// [0, 360) and declination, in degrees (Meeus 21.4).
//
// Proper motion is not applied; this is the precession of the equinoxes alone.
// A caller with a proper motion advances the catalog position first, as the
// book's example 21.b does.
func PrecessEq(raDeg, decDeg, epochFrom, epochTo float64) (outRA, outDec float64) {
	zeta, z, theta := precessionAngles(epochFrom, epochTo)
	sinTheta, cosTheta := math.Sincos(theta)

	sinDec, cosDec := math.Sincos(radians(decDeg))
	sinRAZeta, cosRAZeta := math.Sincos(radians(raDeg) + zeta)

	a := cosDec * sinRAZeta
	b := cosTheta*cosDec*cosRAZeta - sinTheta*sinDec
	c := sinTheta*cosDec*cosRAZeta + cosTheta*sinDec

	outRA = pmod(degrees(math.Atan2(a, b)+z), 360)
	if math.Abs(c) < cosSmallAngle {
		outDec = degrees(math.Asin(c))
		return
	}
	// Within ten arc minutes of the pole the arc sine of c is ill-conditioned,
	// so take the declination from the hypotenuse of a and b instead, which is
	// the cosine of the declination (Meeus, p. 135).
	outDec = degrees(math.Acos(math.Hypot(a, b)))
	if c < 0 {
		outDec = -outDec
	}
	return
}

// precessionAngles returns the three precession angles zeta, z, and theta, in
// radians, for precession between two Julian-year epochs.
func precessionAngles(epochFrom, epochTo float64) (zeta, z, theta float64) {
	// Coefficients of the quadratic in t, the centuries from epochFrom to
	// epochTo. The J2000.0 case is (21.3); anything else needs the T-dependent
	// coefficients of (21.2).
	zetaC := [3]float64{arcsec(2306.2181), arcsec(0.30188), arcsec(0.017998)}
	zC := [3]float64{arcsec(2306.2181), arcsec(1.09468), arcsec(0.018203)}
	thetaC := [3]float64{arcsec(2004.3109), arcsec(-0.42665), arcsec(-0.041833)}
	if epochFrom != 2000 {
		bigT := (epochFrom - 2000) * .01
		zetaC = [3]float64{
			arcsec(Horner(bigT, zetaT[0], zetaT[1], zetaT[2])),
			arcsec(0.30188 - 0.000344*bigT),
			arcsec(0.017998),
		}
		zC = [3]float64{
			arcsec(Horner(bigT, zT[0], zT[1], zT[2])),
			arcsec(1.09468 + 0.000066*bigT),
			arcsec(0.018203),
		}
		thetaC = [3]float64{
			arcsec(Horner(bigT, thetaT[0], thetaT[1], thetaT[2])),
			arcsec(-0.42665 - 0.000217*bigT),
			arcsec(-0.041833),
		}
	}
	t := (epochTo - epochFrom) * .01
	zeta = Horner(t, zetaC[0], zetaC[1], zetaC[2]) * t
	z = Horner(t, zC[0], zC[1], zC[2]) * t
	theta = Horner(t, thetaC[0], thetaC[1], thetaC[2]) * t
	return
}
