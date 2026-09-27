package planetary

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

	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

// lightTimeDaysPerAU is the light-time for one astronomical unit, in days
// (Meeus 33.3).
const lightTimeDaysPerAU = 0.0057755183

// Geometry is the reduction of Meeus chapter 33 at one instant: the planet's
// apparent geocentric ecliptic place and the distances and angles the phase
// and the magnitude need.
type Geometry struct {
	// JDE is the Julian ephemeris day of the observation.
	JDE float64
	// Lon and Lat are the apparent geocentric ecliptic place, true equinox of
	// date, in degrees.
	Lon, Lat float64
	// LonGeom and LatGeom are the place before the FK5 correction and
	// nutation, which the ring tilt and the sub-Earth latitude use.
	LonGeom, LatGeom float64
	// Delta is Earth to planet, R Sun to planet, and EarthR Sun to Earth, in
	// astronomical units.
	Delta, R, EarthR float64
	// Tau is the light-time in days.
	Tau float64
	// Alpha is the phase angle in degrees.
	Alpha float64
}

// geometryAt reduces the planet to its apparent geocentric place at jde.
// Evaluating the Earth, as well as the planet, at the instant the light left
// the planet folds the aberration in with the light-time (Meeus, chapter 33).
func (s *Spec) geometryAt(jde float64) Geometry {
	_, _, earthR := ephemeris.EarthHeliocentric(jde)
	var g Geometry
	g.JDE = jde
	tau := 0.0
	var x, y, z, r float64
	for range 5 {
		l, b, rr := s.Series.Heliocentric(jde - tau)
		l0, b0, r0 := ephemeris.EarthHeliocentric(jde - tau)
		x, y, z = difference(l, b, rr, l0, b0, r0)
		r = rr
		next := lightTimeDaysPerAU * math.Sqrt(x*x+y*y+z*z)
		if math.Abs(next-tau) < 1e-9 {
			tau = next
			break
		}
		tau = next
	}
	g.Tau, g.R, g.EarthR = tau, r, earthR
	// The distance, and the phase and elongation that come from it, are from
	// the Earth at jde to the planet where the light left it. The Earth at
	// jde - tau above only carries the aberration into the direction.
	{
		l, b, rr := s.Series.Heliocentric(jde - tau)
		l0, b0, r0 := ephemeris.EarthHeliocentric(jde)
		dx, dy, dz := difference(l, b, rr, l0, b0, r0)
		g.Delta = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	// The light-time is |planet(jde - tau) - Earth(jde - tau)| / c here, which
	// differs from the exact |planet(jde - tau) - Earth(jde)| / c by v/c of
	// itself, a second or two for Neptune: nothing a planet moves in.
	lon := angles.Normalize(angles.RadToDeg(math.Atan2(y, x)))
	lat := angles.RadToDeg(math.Atan2(z, math.Hypot(x, y)))
	g.LonGeom, g.LatGeom = lon, lat
	dl, db := ephemeris.FK5Correction(lon, lat, jde)
	dpsi, _ := ephemeris.Nutation(jde)
	g.Lon = angles.Normalize(lon + dl + dpsi)
	g.Lat = lat + db
	// Phase angle from the triangle Sun, Earth, planet (Meeus 41.2).
	cosI := (g.R*g.R + g.Delta*g.Delta - g.EarthR*g.EarthR) / (2 * g.R * g.Delta)
	g.Alpha = angles.RadToDeg(math.Acos(Clamp(cosI)))
	return g
}

// difference returns the rectangular vector from the second heliocentric
// place to the first, in astronomical units.
func difference(l, b, r, l0, b0, r0 float64) (x, y, z float64) {
	lr, br := angles.DegToRad(l), angles.DegToRad(b)
	l0r, b0r := angles.DegToRad(l0), angles.DegToRad(b0)
	x = r*math.Cos(br)*math.Cos(lr) - r0*math.Cos(b0r)*math.Cos(l0r)
	y = r*math.Cos(br)*math.Sin(lr) - r0*math.Cos(b0r)*math.Sin(l0r)
	z = r*math.Sin(br) - r0*math.Sin(b0r)
	return x, y, z
}

// Clamp limits x to [-1, 1] before an inverse cosine or sine.
func Clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
