package uranus

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
	"github.com/Bugs5382/go-astronomy/internal/planetary"
)

// Uranus's north pole (IAU, Archinal et al. 2018), J2000 right ascension and
// declination in degrees.
const (
	poleRA  = 257.311
	poleDec = -15.175
)

// subEarthLatitude returns the planetocentric latitude of the sub-Earth point,
// in degrees, from Uranus's geometric geocentric place. The place is of date
// and the pole J2000; the precession between them moves the latitude by well
// under a degree, below what the magnitude term can see.
func subEarthLatitude(g planetary.Geometry) float64 {
	ra, dec := ephemeris.EclToEq(g.LonGeom, g.LatGeom, ephemeris.MeanObliquity(g.JDE))
	a0, d0 := angles.DegToRad(poleRA), angles.DegToRad(poleDec)
	a, d := angles.DegToRad(ra), angles.DegToRad(dec)
	s := -math.Sin(d0)*math.Sin(d) - math.Cos(d0)*math.Cos(d)*math.Cos(a0-a)
	return angles.RadToDeg(math.Asin(planetary.Clamp(s)))
}
