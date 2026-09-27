package sky

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

	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/iau"
)

// vec is a rectangular vector; positions are in astronomical units.
type vec [3]float64

func (a vec) add(b vec) vec       { return vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a vec) sub(b vec) vec       { return vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a vec) scale(k float64) vec { return vec{a[0] * k, a[1] * k, a[2] * k} }
func (a vec) dot(b vec) float64   { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a vec) norm() float64       { return math.Sqrt(a.dot(a)) }
func (a vec) unit() vec           { return a.scale(1 / a.norm()) }

// mat is a 3x3 rotation matrix, applied as m.apply(v) = m v.
type mat [3][3]float64

func (m mat) apply(v vec) vec {
	return vec{
		m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2],
		m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2],
		m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2],
	}
}

func (m mat) mul(n mat) mat {
	var out mat
	for i := range 3 {
		for j := range 3 {
			for k := range 3 {
				out[i][j] += m[i][k] * n[k][j]
			}
		}
	}
	return out
}

func (m mat) transpose() mat {
	var out mat
	for i := range 3 {
		for j := range 3 {
			out[i][j] = m[j][i]
		}
	}
	return out
}

// rot1, rot2, and rot3 rotate the axes by the angle a (radians) about x, y,
// and z: the frame turns, so a fixed vector's coordinates turn by -a.
func rot1(a float64) mat {
	s, c := math.Sincos(a)
	return mat{{1, 0, 0}, {0, c, s}, {0, -s, c}}
}

func rot2(a float64) mat {
	s, c := math.Sincos(a)
	return mat{{c, 0, -s}, {0, 1, 0}, {s, 0, c}}
}

func rot3(a float64) mat {
	s, c := math.Sincos(a)
	return mat{{c, s, 0}, {-s, c, 0}, {0, 0, 1}}
}

const (
	radPerDeg    = math.Pi / 180
	arcsecToRad  = radPerDeg / 3600
	kmPerAU      = ephemeris.KmPerAU
	lightDayAU   = 173.1446326846693 // astronomical units light travels in a day
	j2000        = 2451545.0
	daysPerCentu = 36525.0
)

// ofDateToJ2000 returns the rotation from the mean ecliptic and equinox of
// the Julian ephemeris day jde, the frame of VSOP87D and the Meeus Moon, to
// the J2000 mean equator and equinox, which stands in for the ICRF here (they
// differ by under 0.1 arc second). It is the ecliptic-to-equator turn by the
// mean obliquity of date followed by the IAU 1976 precession back to J2000
// (Meeus 21.2 with a starting epoch of J2000.0).
func ofDateToJ2000(jde float64) mat {
	t := (jde - j2000) / daysPerCentu
	zeta := (2306.2181*t + 0.30188*t*t + 0.017998*t*t*t) * arcsecToRad
	z := (2306.2181*t + 1.09468*t*t + 0.018203*t*t*t) * arcsecToRad
	theta := (2004.3109*t - 0.42665*t*t - 0.041833*t*t*t) * arcsecToRad
	// J2000 equator to the equator of date is R3(-z) R2(theta) R3(-zeta).
	precess := rot3(-z).mul(rot2(theta)).mul(rot3(-zeta))
	// Ecliptic of date to the equator of date: turn by -epsilon about x.
	eclToEq := rot1(-ephemeris.MeanObliquity(jde) * radPerDeg)
	return precess.transpose().mul(eclToEq)
}

// spherical returns the rectangular vector of an ecliptic longitude and
// latitude in degrees and a distance.
func spherical(lonDeg, latDeg, r float64) vec {
	sl, cl := math.Sincos(lonDeg * radPerDeg)
	sb, cb := math.Sincos(latDeg * radPerDeg)
	return vec{r * cb * cl, r * cb * sl, r * sb}
}

// bodyFixed returns the rotation from the J2000 equator (ICRF) to the body's
// frame at orientation o: z along the north pole, x through the prime
// meridian, y 90 degrees east of it.
func bodyFixed(o iau.Orientation) mat {
	return rot3(o.W * radPerDeg).mul(rot1((90 - o.Delta0) * radPerDeg)).mul(rot3((90 + o.Alpha0) * radPerDeg))
}

// geodetic returns a site's position in the body's frame, in km, from its
// planetodetic latitude and east longitude in degrees and its height above
// the reference ellipsoid in km, together with the local up, east, and north
// unit vectors.
func geodetic(m *iau.Model, latDeg, lonDeg, heightKm float64) (pos, up, east, north vec) {
	sp, cp := math.Sincos(latDeg * radPerDeg)
	sl, cl := math.Sincos(lonDeg * radPerDeg)
	a, b := m.EquatorialKm, m.PolarKm
	e2 := 1 - (b*b)/(a*a)
	n := a / math.Sqrt(1-e2*sp*sp)
	pos = vec{(n + heightKm) * cp * cl, (n + heightKm) * cp * sl, (n*(1-e2) + heightKm) * sp}
	up = vec{cp * cl, cp * sl, sp}
	east = vec{-sl, cl, 0}
	north = vec{-sp * cl, -sp * sl, cp}
	return pos, up, east, north
}
