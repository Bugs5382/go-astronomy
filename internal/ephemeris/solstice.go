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

// Equinoxes and solstices.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 27,
// "Equinoxes and Solstices": tables 27.A and 27.B for the mean instant, and
// table 27.C with the correction of p. 178 for the true instant. The Go form
// derives from soniakeys/meeus (solstice), MIT licensed.
//
// The series gives the instant to within one minute for the years 1951 to
// 2050. It is then refined by the higher-accuracy method of p. 180: the Sun's
// apparent longitude from VSOP87 is evaluated at the estimate, and the
// estimate is moved by 58 sin(k 90 - lambda) days until the correction
// vanishes, which lands on the exact quadrant crossing (issue 45).

import "math"

// Mean equinox and solstice coefficients, in Julian ephemeris days, as
// polynomials in Y. Table 27.A takes Y as the year divided by 1000 and covers
// the years -1000 to +1000; table 27.B takes Y as the year less 2000, divided
// by 1000, and covers +1000 to +3000.
var (
	marchAncient     = []float64{1721139.29189, 365242.13740, .06134, .00111, -.00071}
	juneAncient      = []float64{1721233.25401, 365241.72562, -.05232, .00907, .00025}
	septemberAncient = []float64{1721325.70455, 365242.49558, -.11677, -.00297, .00074}
	decemberAncient  = []float64{1721414.39987, 365242.88257, -.00769, -.00933, -.00006}

	marchModern     = []float64{2451623.80984, 365242.37404, .05169, -.00411, -.00057}
	juneModern      = []float64{2451716.56767, 365241.62603, .00325, .00888, -.00030}
	septemberModern = []float64{2451810.21715, 365242.01767, -.11575, .00337, .00078}
	decemberModern  = []float64{2451900.05952, 365242.74049, -.06223, -.00823, .00032}
)

// seasonTerms is table 27.C: the twenty-four periodic terms that correct the
// mean instant to the true one. Amplitude is in units of 0.00001 day; the phase
// and frequency are degrees and degrees per Julian century.
var seasonTerms = [...]struct{ amplitude, phase, frequency float64 }{
	{485, 324.96, 1934.136},
	{203, 337.23, 32964.467},
	{199, 342.08, 20.186},
	{182, 27.85, 445267.112},
	{156, 73.14, 45036.886},
	{136, 171.52, 22518.443},
	{77, 222.54, 65928.934},
	{74, 296.72, 3034.906},
	{70, 243.58, 9037.513},
	{58, 119.81, 33718.147},
	{52, 297.17, 150.678},
	{50, 21.02, 2281.226},
	{45, 247.54, 29929.562},
	{44, 325.15, 31555.956},
	{29, 60.93, 4443.417},
	{18, 155.12, 67555.328},
	{17, 288.79, 4562.452},
	{16, 198.04, 62894.029},
	{14, 199.76, 31436.921},
	{12, 95.39, 14577.848},
	{12, 287.11, 31931.756},
	{12, 320.81, 34777.259},
	{9, 227.73, 1222.114},
	{8, 15.45, 16859.074},
}

// MarchEquinox returns the Julian ephemeris day of the March equinox of the
// given year, the instant the Sun's apparent longitude reaches 0 degrees.
func MarchEquinox(year int) float64 {
	return refineSeason(seasonBoundary(year, marchAncient, marchModern), 0)
}

// JuneSolstice returns the Julian ephemeris day of the June solstice of the
// given year, the instant the Sun's apparent longitude reaches 90 degrees.
func JuneSolstice(year int) float64 {
	return refineSeason(seasonBoundary(year, juneAncient, juneModern), 90)
}

// SeptemberEquinox returns the Julian ephemeris day of the September equinox of
// the given year, the instant the Sun's apparent longitude reaches 180 degrees.
func SeptemberEquinox(year int) float64 {
	return refineSeason(seasonBoundary(year, septemberAncient, septemberModern), 180)
}

// DecemberSolstice returns the Julian ephemeris day of the December solstice of
// the given year, the instant the Sun's apparent longitude reaches 270 degrees.
func DecemberSolstice(year int) float64 {
	return refineSeason(seasonBoundary(year, decemberAncient, decemberModern), 270)
}

// seasonBoundary evaluates one equinox or solstice, choosing between the
// ancient and modern coefficient sets at the year 1000 as the book directs.
func seasonBoundary(year int, ancient, modern []float64) float64 {
	if year < 1000 {
		return seasonCorrect(float64(year)*.001, ancient)
	}
	return seasonCorrect(float64(year-2000)*.001, modern)
}

// seasonCorrect returns the true instant: the mean instant from the given
// coefficients corrected by the periodic terms of table 27.C, scaled by the
// Sun's apparent motion so the correction is a time rather than a longitude
// (Meeus, p. 178).
func seasonCorrect(y float64, mean []float64) float64 {
	j0 := Horner(y, mean...)
	t := J2000Century(j0)
	w := radians(35999.373*t - 2.47)
	// The Sun's apparent daily motion relative to its mean motion, which turns
	// the longitude correction into a correction in time.
	dLambda := 1 + .0334*math.Cos(w) + .0007*math.Cos(2*w)
	// Sum from the end so the smallest terms accumulate first.
	var s float64
	for i := len(seasonTerms) - 1; i >= 0; i-- {
		term := &seasonTerms[i]
		s += term.amplitude * math.Cos(radians(term.phase+term.frequency*t))
	}
	return j0 + .00001*s/dLambda
}

// refineSeason moves an estimate of the instant the Sun's apparent longitude
// reaches target degrees onto the crossing itself (Meeus, p. 180). Each step
// is 58 sin(target - lambda) days, where 58 days is roughly the Sun's motion
// of one radian divided into the year; from an estimate within a minute the
// iteration settles in two or three steps.
func refineSeason(jde, target float64) float64 {
	for range 10 {
		lon, _, _ := SunApparentEcliptic(jde)
		step := 58 * math.Sin(radians(target-lon))
		jde += step
		if math.Abs(step) < 1e-7 {
			break
		}
	}
	return jde
}
