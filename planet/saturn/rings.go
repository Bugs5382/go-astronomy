package saturn

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

// ringTilt returns the Saturnicentric latitude of the Earth referred to the
// ring plane, in degrees, from Saturn's geometric geocentric ecliptic place
// (lonDeg, latDeg, of date) at the Julian ephemeris day jde (Meeus 45.3, with
// the ring plane's inclination and node of 45.1 and 45.2). Positive means the
// Earth sees the north face of the rings.
func ringTilt(lonDeg, latDeg, jde float64) float64 {
	t := ephemeris.J2000Century(jde)
	i := angles.DegToRad(28.075216 - 0.012998*t + 0.000004*t*t)
	node := 169.508470 + 1.394681*t + 0.000412*t*t
	lam := angles.DegToRad(lonDeg - node)
	beta := angles.DegToRad(latDeg)
	sinB := math.Sin(i)*math.Cos(beta)*math.Sin(lam) - math.Cos(i)*math.Sin(beta)
	return angles.RadToDeg(math.Asin(sinB))
}
