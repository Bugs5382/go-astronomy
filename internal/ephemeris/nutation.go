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

// Nutation, the obliquity of the ecliptic, and sidereal time at Greenwich.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed.: chapter 22,
// "Nutation and the Obliquity of the Ecliptic", including the abridged IAU 1980
// series of table 22.A and formula (22.2) for the mean obliquity; and
// chapter 12, "Sidereal Time at Greenwich", formula (12.2). The Go form derives
// from soniakeys/meeus (nutation, sidereal), MIT licensed.

import "math"

// arcsecPerDeg and secPerDeg convert the units the book prints its series in.
// One degree is 3600 arc seconds of angle and, as an hour angle, 240 seconds of
// time.
const (
	arcsecPerDeg = 3600.0
	secPerDeg    = 240.0
)

// obliquityJ2000 is the mean obliquity of the ecliptic at J2000.0, 23d26'21.448"
// as decimal degrees. It is the constant term of formula (22.2).
const obliquityJ2000 = 23 + 26.0/60 + 21.448/3600

// pmod returns x modulo y with the sign of y, so the result of a reduction into
// [0, y) is never negative. math.Mod alone keeps the sign of x.
func pmod(x, y float64) float64 {
	r := math.Mod(x, y)
	if r < 0 {
		r += y
	}
	// A tiny negative x can round up to exactly y after the adjustment; keep
	// the range half-open.
	if r >= y {
		r -= y
	}
	return r
}

// Nutation returns the nutation in longitude and the nutation in obliquity, in
// degrees, at the given Julian ephemeris day.
//
// Computation is by the IAU 1980 theory of nutation abridged to the 63 terms of
// Meeus table 22.A, which drops terms below 0.0003 arc seconds.
func Nutation(jde float64) (dpsi, deps float64) {
	t := J2000Century(jde)
	// The five fundamental arguments, p. 144. Each is a polynomial in T with a
	// rational final coefficient: those must be written with a decimal point,
	// because 1/189474 is an integer expression in Go and evaluates to zero.
	d := radians(Horner(t, 297.85036, 445267.11148, -0.0019142, 1.0/189474))
	m := radians(Horner(t, 357.52772, 35999.050340, -0.0001603, -1.0/300000))
	mp := radians(Horner(t, 134.96298, 477198.867398, 0.0086972, 1.0/5620))
	f := radians(Horner(t, 93.27191, 483202.017538, -0.0036825, 1.0/327270))
	omega := radians(Horner(t, 125.04452, -1934.136261, 0.0020708, 1.0/450000))

	// Accumulate from the end of the table so the smallest terms are added
	// first and less of their significance is lost.
	var sumPsi, sumEps float64
	for i := len(nutationTerms) - 1; i >= 0; i-- {
		r := &nutationTerms[i]
		arg := r.d*d + r.m*m + r.mp*mp + r.f*f + r.omega*omega
		s, c := math.Sincos(arg)
		sumPsi += s * (r.psi0 + r.psi1*t)
		sumEps += c * (r.eps0 + r.eps1*t)
	}
	// Table coefficients are in units of 0.0001 arc second.
	return sumPsi * 1e-4 / arcsecPerDeg, sumEps * 1e-4 / arcsecPerDeg
}

// Nutated is the nutation and the obliquity of the ecliptic at one instant.
// The 63-term nutation series is most of the cost of an apparent place, and a
// Sun position needs it three times (the apparent longitude, the true
// obliquity, and the apparent sidereal time); NutationAt evaluates it once for
// all of them. Every method gives exactly the value of the function it stands
// for.
type Nutated struct {
	// DPsi and DEps are the nutation in longitude and in obliquity, and
	// MeanEps the mean obliquity, all in degrees.
	DPsi, DEps, MeanEps float64
}

// NutationAt returns the nutation and the mean obliquity at the Julian
// ephemeris day jde.
func NutationAt(jde float64) Nutated {
	dpsi, deps := Nutation(jde)
	return Nutated{DPsi: dpsi, DEps: deps, MeanEps: MeanObliquity(jde)}
}

// TrueObliquity is TrueObliquity(jde) for the instant of n.
func (n Nutated) TrueObliquity() float64 { return n.MeanEps + n.DEps }

// InRA is NutationInRA(jde) for the instant of n.
func (n Nutated) InRA() float64 { return n.DPsi * math.Cos(radians(n.MeanEps+n.DEps)) }

// MeanObliquity returns the mean obliquity of the ecliptic, in degrees, at the
// given Julian ephemeris day, by the IAU 1980 polynomial (Meeus 22.2).
//
// Accuracy is 1 arc second over the years 1000 to 3000 and 10 arc seconds over
// the years 0 to 4000, which is where this library operates. Meeus also gives
// Laskar's longer-range polynomial (22.3); it is not implemented because it is
// no more accurate inside that window and this library does not claim results
// outside it.
func MeanObliquity(jde float64) float64 {
	return Horner(J2000Century(jde),
		obliquityJ2000,
		-46.8150/arcsecPerDeg,
		-0.00059/arcsecPerDeg,
		0.001813/arcsecPerDeg)
}

// TrueObliquity returns the true obliquity of the ecliptic, in degrees: the
// mean obliquity plus the nutation in obliquity.
func TrueObliquity(jde float64) float64 {
	_, deps := Nutation(jde)
	return MeanObliquity(jde) + deps
}

// NutationInRA returns the nutation in right ascension, in degrees, also called
// the equation of the equinoxes: the amount by which apparent sidereal time
// exceeds mean sidereal time (Meeus, chapter 12, p. 88).
func NutationInRA(jde float64) float64 {
	dpsi, deps := Nutation(jde)
	return dpsi * math.Cos(radians(MeanObliquity(jde)+deps))
}

// iau1982Sidereal are the coefficients, in seconds of time, of mean sidereal
// time at Greenwich at 0h UT as a polynomial in centuries from J2000.0. They
// are the values adopted by the International Astronomical Union in 1982 and
// printed in Meeus (12.2).
var iau1982Sidereal = []float64{24110.54841, 8640184.812866, 0.093104, 0.0000062}

// siderealRate is the ratio of a mean sidereal day to a mean solar day: the
// factor by which sidereal time runs fast (Meeus, chapter 12, p. 87).
const siderealRate = 1.00273790935

// MeanSiderealTime returns mean sidereal time at Greenwich, in degrees in
// [0, 360), for the given Julian day (Meeus 12.2).
//
// The polynomial is evaluated at 0h UT of the day containing jd and the elapsed
// fraction of the day is then added at the sidereal rate, which is how the book
// arranges the computation and avoids evaluating the large linear term at a
// fractional argument.
func MeanSiderealTime(jd float64) float64 {
	seconds, dayFrac := meanSidereal0UT(jd)
	return pmod(seconds+dayFrac*86400*siderealRate, 86400) / 3600 * 15
}

// ApparentSiderealTime returns apparent sidereal time at Greenwich, in degrees
// in [0, 360): mean sidereal time plus the nutation in right ascension.
func ApparentSiderealTime(jd float64) float64 {
	seconds, dayFrac := meanSidereal0UT(jd)
	mean := seconds + dayFrac*86400*siderealRate
	return pmod(mean+NutationInRA(jd)*secPerDeg, 86400) / 3600 * 15
}

// meanSidereal0UT returns mean sidereal time at Greenwich at 0h UT of the day
// containing jd, in seconds of time and without reduction to one revolution,
// together with the fraction of that day elapsed at jd.
func meanSidereal0UT(jd float64) (seconds, dayFrac float64) {
	// A Julian day begins at noon, so shifting by half a day puts the integer
	// part at midnight, which is the epoch the polynomial is defined for.
	j0, f := math.Modf(jd + .5)
	return Horner(J2000Century(j0-.5), iau1982Sidereal...), f
}

// radians converts degrees to radians. It is spelled out here so this package
// stays free of dependencies, including on the rest of this module.
func radians(deg float64) float64 { return deg * math.Pi / 180 }

// degrees converts radians to degrees.
func degrees(rad float64) float64 { return rad * 180 / math.Pi }

// nutationTerms is table 22.A: the periodic terms of nutation in longitude and
// obliquity. The first five columns are the integer multiples of the
// fundamental arguments D, M, M', F, and Omega. The last four are the
// coefficients of the sine series for nutation in longitude and the cosine
// series for nutation in obliquity, each with a linear term in T, in units of
// 0.0001 arc second.
var nutationTerms = [...]struct {
	d, m, mp, f, omega     float64
	psi0, psi1, eps0, eps1 float64
}{
	{0, 0, 0, 0, 1, -171996, -174.2, 92025, 8.9},
	{-2, 0, 0, 2, 2, -13187, -1.6, 5736, -3.1},
	{0, 0, 0, 2, 2, -2274, -0.2, 977, -0.5},
	{0, 0, 0, 0, 2, 2062, 0.2, -895, 0.5},
	{0, 1, 0, 0, 0, 1426, -3.4, 54, -0.1},
	{0, 0, 1, 0, 0, 712, 0.1, -7, 0},
	{-2, 1, 0, 2, 2, -517, 1.2, 224, -0.6},
	{0, 0, 0, 2, 1, -386, -0.4, 200, 0},
	{0, 0, 1, 2, 2, -301, 0, 129, -0.1},
	{-2, -1, 0, 2, 2, 217, -0.5, -95, 0.3},
	{-2, 0, 1, 0, 0, -158, 0, 0, 0},
	{-2, 0, 0, 2, 1, 129, 0.1, -70, 0},
	{0, 0, -1, 2, 2, 123, 0, -53, 0},
	{2, 0, 0, 0, 0, 63, 0, 0, 0},
	{0, 0, 1, 0, 1, 63, 0.1, -33, 0},
	{2, 0, -1, 2, 2, -59, 0, 26, 0},
	{0, 0, -1, 0, 1, -58, -0.1, 32, 0},
	{0, 0, 1, 2, 1, -51, 0, 27, 0},
	{-2, 0, 2, 0, 0, 48, 0, 0, 0},
	{0, 0, -2, 2, 1, 46, 0, -24, 0},
	{2, 0, 0, 2, 2, -38, 0, 16, 0},
	{0, 0, 2, 2, 2, -31, 0, 13, 0},
	{0, 0, 2, 0, 0, 29, 0, 0, 0},
	{-2, 0, 1, 2, 2, 29, 0, -12, 0},
	{0, 0, 0, 2, 0, 26, 0, 0, 0},
	{-2, 0, 0, 2, 0, -22, 0, 0, 0},
	{0, 0, -1, 2, 1, 21, 0, -10, 0},
	{0, 2, 0, 0, 0, 17, -0.1, 0, 0},
	{2, 0, -1, 0, 1, 16, 0, -8, 0},
	{-2, 2, 0, 2, 2, -16, 0.1, 7, 0},
	{0, 1, 0, 0, 1, -15, 0, 9, 0},
	{-2, 0, 1, 0, 1, -13, 0, 7, 0},
	{0, -1, 0, 0, 1, -12, 0, 6, 0},
	{0, 0, 2, -2, 0, 11, 0, 0, 0},
	{2, 0, -1, 2, 1, -10, 0, 5, 0},
	{2, 0, 1, 2, 2, -8, 0, 3, 0},
	{0, 1, 0, 2, 2, 7, 0, -3, 0},
	{-2, 1, 1, 0, 0, -7, 0, 0, 0},
	{0, -1, 0, 2, 2, -7, 0, 3, 0},
	{2, 0, 0, 2, 1, -7, 0, 3, 0},
	{2, 0, 1, 0, 0, 6, 0, 0, 0},
	{-2, 0, 2, 2, 2, 6, 0, -3, 0},
	{-2, 0, 1, 2, 1, 6, 0, -3, 0},
	{2, 0, -2, 0, 1, -6, 0, 3, 0},
	{2, 0, 0, 0, 1, -6, 0, 3, 0},
	{0, -1, 1, 0, 0, 5, 0, 0, 0},
	{-2, -1, 0, 2, 1, -5, 0, 3, 0},
	{-2, 0, 0, 0, 1, -5, 0, 3, 0},
	{0, 0, 2, 2, 1, -5, 0, 3, 0},
	{-2, 0, 2, 0, 1, 4, 0, 0, 0},
	{-2, 1, 0, 2, 1, 4, 0, 0, 0},
	{0, 0, 1, -2, 0, 4, 0, 0, 0},
	{-1, 0, 1, 0, 0, -4, 0, 0, 0},
	{-2, 1, 0, 0, 0, -4, 0, 0, 0},
	{1, 0, 0, 0, 0, -4, 0, 0, 0},
	{0, 0, 1, 2, 0, 3, 0, 0, 0},
	{0, 0, -2, 2, 2, -3, 0, 0, 0},
	{-1, -1, 1, 0, 0, -3, 0, 0, 0},
	{0, 1, 1, 0, 0, -3, 0, 0, 0},
	{0, -1, 1, 2, 2, -3, 0, 0, 0},
	{2, -1, -1, 2, 2, -3, 0, 0, 0},
	{0, 0, 3, 2, 2, -3, 0, 0, 0},
	{2, -1, 0, 2, 2, -3, 0, 0, 0},
}
