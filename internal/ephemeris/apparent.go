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

// The apparent place of a star: the direction of a body so far away that it
// has no parallax, from its J2000 mean place to the true equator and equinox
// of date. Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 23, does it
// with closed-form corrections; here the same steps are rotations of a unit
// vector: the IAU 1976 precession (21.2 with a J2000.0 starting epoch), the
// IAU 1980 nutation, and the annual aberration from the Earth's velocity,
// which VSOP87 gives by differencing (the book's 23.2 is its first-order form).

import "math"

// lightDayAU is the distance light travels in a day, in astronomical units.
const lightDayAU = 173.1446326846693

// velocityHalfStep is the half-interval, in days, of the central difference
// that gives the Earth's heliocentric velocity for the aberration.
const velocityHalfStep = 0.005

// ApparentFromJ2000 returns the apparent right ascension, in degrees in
// [0, 360), and declination, in degrees, on the true equator and equinox of
// the Julian ephemeris day jde, of a star at J2000 mean place (raDeg, decDeg),
// with dpsi and deps the nutation in longitude and obliquity at jde, in
// degrees (see Nutation), and distancePc the star's distance in parsecs, or
// zero when unknown. Proper motion, if any, is the caller's: pass the mean
// place already carried to the epoch. For many stars at one instant, build
// the ApparentFrame once instead.
func ApparentFromJ2000(raDeg, decDeg, jde, dpsi, deps, distancePc float64) (outRA, outDec float64) {
	return NewApparentFrame(jde, dpsi, deps).Apply(raDeg, decDeg, distancePc)
}

// ApparentFrame is everything the apparent place of a star needs that does
// not depend on the star: the rotation from the J2000 mean equator to the
// true equator and equinox of date, and the Earth's heliocentric position and
// velocity in that frame. Building one costs a few microseconds (three
// VSOP87 evaluations and the rotations); applying it to a star costs a
// fraction of that, so a sky of thousands of stars builds one per instant.
//
// Apply precesses (IAU 1976, Meeus 21.2 with a J2000.0 starting epoch),
// nutates (IAU 1980), applies the annual parallax of a star with a distance
// (0.75 arc second for Alpha Centauri), and the annual aberration of the
// Earth's heliocentric motion (the barycentric motion differs by 0.01 arc
// second). The gravitational bending of light by the Sun (under 0.01 arc
// second more than 45 degrees from it) and the diurnal aberration of the
// observer's rotation (up to 0.3 arc second) are not applied.
type ApparentFrame struct {
	rot      [3][3]float64 // J2000 mean equator to the true equator of date
	earthAU  [3]float64    // the Earth's heliocentric position, true equator of date
	velocity [3]float64    // the Earth's heliocentric velocity, in units of c
}

// NewApparentFrame builds the frame at the Julian ephemeris day jde, with
// dpsi and deps the nutation there, in degrees.
func NewApparentFrame(jde, dpsi, deps float64) ApparentFrame {
	t := J2000Century(jde)
	zeta := arcsec((2306.2181 + (0.30188+0.017998*t)*t) * t)
	z := arcsec((2306.2181 + (1.09468+0.018203*t)*t) * t)
	theta := arcsec((2004.3109 - (0.42665+0.041833*t)*t) * t)
	meanEps := MeanObliquity(jde)
	// J2000 mean equator to the mean equator of date, then to the true
	// equator of date.
	toDate := func(w [3]float64) [3]float64 {
		w = rotZ(-z, rotY(theta, rotZ(-zeta, w)))
		return rotX(-radians(meanEps+deps), rotZ(-radians(dpsi), rotX(radians(meanEps), w)))
	}
	// From the ecliptic of date to the true equator of date: the nutation in
	// longitude, then the true obliquity.
	eclToTrue := func(w [3]float64) [3]float64 { return rotX(-radians(meanEps+deps), rotZ(-radians(dpsi), w)) }

	var f ApparentFrame
	for j, e := range [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} {
		col := toDate(e)
		for i := range 3 {
			f.rot[i][j] = col[i]
		}
	}
	f.earthAU = eclToTrue(earthRectangular(jde))
	v := eclToTrue(earthVelocityEcliptic(jde))
	f.velocity = [3]float64{v[0] / lightDayAU, v[1] / lightDayAU, v[2] / lightDayAU}
	return f
}

// Apply returns the apparent right ascension, in degrees in [0, 360), and
// declination, in degrees, of a star at J2000 mean place (raDeg, decDeg) and
// distancePc parsecs (zero when unknown).
func (f ApparentFrame) Apply(raDeg, decDeg, distancePc float64) (outRA, outDec float64) {
	m := unitVector(raDeg, decDeg)
	u := [3]float64{
		f.rot[0][0]*m[0] + f.rot[0][1]*m[1] + f.rot[0][2]*m[2],
		f.rot[1][0]*m[0] + f.rot[1][1]*m[1] + f.rot[1][2]*m[2],
		f.rot[2][0]*m[0] + f.rot[2][1]*m[1] + f.rot[2][2]*m[2],
	}
	// Annual parallax: the star seen from the Earth, not the Sun, with the
	// star's distance and the Earth's position both in astronomical units.
	if distancePc > 0 {
		const auPerParsec = 206264.80624709636
		d := distancePc * auPerParsec
		u = [3]float64{d*u[0] - f.earthAU[0], d*u[1] - f.earthAU[1], d*u[2] - f.earthAU[2]}
		n := math.Sqrt(u[0]*u[0] + u[1]*u[1] + u[2]*u[2])
		u = [3]float64{u[0] / n, u[1] / n, u[2] / n}
	}
	// Annual aberration, to first order in v/c.
	u = [3]float64{u[0] + f.velocity[0], u[1] + f.velocity[1], u[2] + f.velocity[2]}
	r := math.Sqrt(u[0]*u[0] + u[1]*u[1] + u[2]*u[2])
	outRA = pmod(degrees(math.Atan2(u[1], u[0])), 360)
	outDec = degrees(math.Asin(u[2] / r))
	return outRA, outDec
}

// earthVelocityEcliptic returns the Earth's heliocentric velocity at jde, in
// astronomical units per day, on the mean ecliptic and equinox of date.
func earthVelocityEcliptic(jde float64) [3]float64 {
	a := earthRectangular(jde + velocityHalfStep)
	b := earthRectangular(jde - velocityHalfStep)
	k := 1 / (2 * velocityHalfStep)
	return [3]float64{(a[0] - b[0]) * k, (a[1] - b[1]) * k, (a[2] - b[2]) * k}
}

// earthRectangular returns the Earth's heliocentric position from VSOP87 as
// rectangular ecliptic coordinates of date, in astronomical units.
func earthRectangular(jde float64) [3]float64 {
	l, b, r := EarthHeliocentric(jde)
	sl, cl := math.Sincos(radians(l))
	sb, cb := math.Sincos(radians(b))
	return [3]float64{r * cb * cl, r * cb * sl, r * sb}
}

// unitVector returns the unit vector of a right ascension and declination in
// degrees.
func unitVector(raDeg, decDeg float64) [3]float64 {
	sa, ca := math.Sincos(radians(raDeg))
	sd, cd := math.Sincos(radians(decDeg))
	return [3]float64{cd * ca, cd * sa, sd}
}

// rotX, rotY, and rotZ turn the axes by a radians about x, y, and z, so the
// coordinates of a fixed vector turn by -a.
func rotX(a float64, v [3]float64) [3]float64 {
	s, c := math.Sincos(a)
	return [3]float64{v[0], c*v[1] + s*v[2], -s*v[1] + c*v[2]}
}

func rotY(a float64, v [3]float64) [3]float64 {
	s, c := math.Sincos(a)
	return [3]float64{c*v[0] - s*v[2], v[1], s*v[0] + c*v[2]}
}

func rotZ(a float64, v [3]float64) [3]float64 {
	s, c := math.Sincos(a)
	return [3]float64{c*v[0] + s*v[1], -s*v[0] + c*v[1], v[2]}
}
