package planet

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

// Apparent magnitudes, from A. Mallama and J. L. Hilton, "Computing apparent
// planetary magnitudes for The Astronomical Almanac", Astronomy and Computing
// 25, 10 (2018). Each planet's magnitude is V(1, alpha) + 5 log10(r delta),
// where alpha is the phase angle in degrees and r and delta are the distances
// from the Sun and from the observer in astronomical units. The small terms
// that depend on Mars's orbital longitude are left out (a few hundredths of a
// magnitude).

import (
	"math"

	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

func magnitude(b Body, g geocentric, alpha, jde float64) float64 {
	return absoluteMagnitude(b, g, alpha, jde) + 5*math.Log10(g.r*g.delta)
}

// absoluteMagnitude returns V(1, alpha), the magnitude at unit distance from
// the Sun and the observer.
func absoluteMagnitude(b Body, g geocentric, a, jde float64) float64 {
	switch b {
	case Mercury:
		return ephemeris.Horner(a, -0.613, 6.3280e-02, -1.6336e-03, 3.3644e-05, -3.4265e-07, 1.6893e-09, -3.0334e-12)
	case Venus:
		if a <= 163.7 {
			return ephemeris.Horner(a, -4.384, -1.044e-03, 3.687e-04, -2.814e-06, 8.938e-09)
		}
		return ephemeris.Horner(a, 236.05828, -2.81914, 8.39034e-03)
	case Mars:
		if a <= 50 {
			return ephemeris.Horner(a, -1.601, 2.267e-02, -1.302e-04)
		}
		return ephemeris.Horner(a, -0.367, -0.02573, 3.445e-04)
	case Jupiter:
		return ephemeris.Horner(a, -9.395, -3.7e-04, 6.16e-04)
	case Saturn:
		// The globe and rings together, for phase angles up to 6.5 degrees
		// and ring tilts up to 27 degrees, which covers every view from the
		// Earth.
		sb := math.Sin(angles.DegToRad(math.Abs(saturnRingTilt(g, jde))))
		return -8.914 - 1.825*sb + 0.026*a - 0.378*sb*math.Exp(-2.25*a)
	case Uranus:
		// The sub-Earth latitude term uses the planetocentric latitude, which
		// differs from the planetographic one by under a degree here.
		phi := math.Abs(subEarthLatitude(g, uranusPoleRA, uranusPoleDec, jde))
		return ephemeris.Horner(a, -7.110, 6.587e-03, 1.045e-04) - 8.4e-04*phi
	case Neptune:
		return ephemeris.Horner(a, -7.00, 7.944e-03, 9.617e-05)
	}
	return math.NaN()
}

// saturnRingTilt returns the Saturnicentric latitude of the Earth referred to
// the ring plane, in degrees (Meeus 45.3, with the ring plane's inclination and
// node of 45.1 and 45.2 referred to the ecliptic and equinox of date).
func saturnRingTilt(g geocentric, jde float64) float64 {
	t := ephemeris.J2000Century(jde)
	i := angles.DegToRad(28.075216 - 0.012998*t + 0.000004*t*t)
	node := 169.508470 + 1.394681*t + 0.000412*t*t
	lam := angles.DegToRad(g.lonGeom - node)
	beta := angles.DegToRad(g.latGeom)
	sinB := math.Sin(i)*math.Cos(beta)*math.Sin(lam) - math.Cos(i)*math.Sin(beta)
	return angles.RadToDeg(math.Asin(sinB))
}

// Uranus's north pole (IAU, Archinal et al. 2018), J2000 right ascension and
// declination in degrees.
const (
	uranusPoleRA  = 257.311
	uranusPoleDec = -15.175
)

// subEarthLatitude returns the planetocentric latitude of the sub-Earth point,
// in degrees, for a planet whose north pole is at poleRA, poleDec (J2000),
// from the planet's geometric geocentric place (Meeus 42.x in its general
// form). The place is of date and the pole J2000; the precession between them
// moves the latitude by well under a degree, below what the magnitude term
// can see.
func subEarthLatitude(g geocentric, poleRA, poleDec, jde float64) float64 {
	ra, dec := ephemeris.EclToEq(g.lonGeom, g.latGeom, ephemeris.MeanObliquity(jde))
	a0, d0 := angles.DegToRad(poleRA), angles.DegToRad(poleDec)
	a, d := angles.DegToRad(ra), angles.DegToRad(dec)
	s := -math.Sin(d0)*math.Sin(d) - math.Cos(d0)*math.Cos(d)*math.Cos(a0-a)
	return angles.RadToDeg(math.Asin(clamp(s)))
}
