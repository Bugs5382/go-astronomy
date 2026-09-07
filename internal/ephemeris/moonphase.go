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

// Illuminated fraction of the Moon's disk, and the instants of New and Full
// Moon.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed.: chapter 48,
// "Illuminated Fraction of the Moon's Disk", formulae (48.1) and the
// low-accuracy (48.4); and chapter 49, "Phases of the Moon", formulae (49.1)
// through (49.3) with the correction tables of pp. 351 and 352. The Go form
// derives from soniakeys/meeus (moonillum, moonphase, base), MIT licensed.
//
// Only New and Full Moon are implemented, which is what this library needs. The
// First and Last Quarter series is a separate table and is left out rather than
// carried untested.

import "math"

// MoonPhaseAngle returns the Sun-Moon-Earth phase angle, in degrees in
// [0, 360), at the given Julian ephemeris day, by the low-accuracy formula
// (Meeus 48.4).
//
// The formula uses only the fundamental arguments, so it needs neither the
// Moon's nor the Sun's position. It is about 0.2 degrees from the accurate
// method, which is a few thousandths in the illuminated fraction it implies.
//
// Fixed against the source this port derives from: the cubic and quartic
// coefficients of the fundamental arguments were written as untyped integer
// divisions, which Go evaluates to exactly zero, so those terms were silently
// absent. The quadratic coefficient of the Sun's mean anomaly is also corrected
// to the -0.0001536 the book gives.
func MoonPhaseAngle(jde float64) float64 {
	t := J2000Century(jde)
	d := pmod(Horner(t, 297.8501921, 445267.1114034, -0.0018819, 1.0/545868, -1.0/113065000), 360)
	m := pmod(Horner(t, 357.5291092, 35999.0502909, -0.0001536, 1.0/24490000), 360)
	mp := pmod(Horner(t, 134.9633964, 477198.8675055, 0.0087414, 1.0/69699, -1.0/14712000), 360)
	dr, mr, mpr := radians(d), radians(m), radians(mp)
	angle := 180 - d +
		-6.289*math.Sin(mpr) +
		2.100*math.Sin(mr) +
		-1.274*math.Sin(2*dr-mpr) +
		-0.658*math.Sin(2*dr) +
		-0.214*math.Sin(2*mpr) +
		-0.110*math.Sin(dr)
	return pmod(angle, 360)
}

// IlluminatedFraction returns the fraction of a body's disk that is lit, in
// [0, 1], given the phase angle in degrees (Meeus 48.1). It is 1 at a phase
// angle of zero, when the disk is fully lit, and 0 at 180 degrees.
func IlluminatedFraction(phaseAngleDeg float64) float64 {
	return (1 + math.Cos(radians(phaseAngleDeg))) * 0.5
}

// lunationsPerJulianYear is the mean number of lunations in a Julian year, the
// factor that maps a decimal year to a lunation index (Meeus 49.2).
const lunationsPerJulianYear = 12.3685

// lunationScale converts a lunation index to the time argument T of the phase
// series: T is the index divided by 1236.85, which is the number of lunations
// in a Julian century (Meeus 49.3).
const lunationScale = 1 / 1236.85

// lunationNumber returns the lunation index of the phase nearest the given
// decimal year. The quarter argument selects the phase within the lunation:
// 0 for New Moon, 0.25 for First Quarter, 0.5 for Full Moon, 0.75 for Last
// Quarter (Meeus 49.2).
func lunationNumber(year, quarter float64) float64 {
	k := (year - 2000) * lunationsPerJulianYear
	return math.Floor(k-quarter+.5) + quarter
}

// meanLunarPhase returns the Julian ephemeris day of the mean phase at lunation
// index k (Meeus 49.1). The mean instant is within half a day of the true one.
func meanLunarPhase(k float64) float64 {
	t := k * lunationScale
	// The linear term is written as a coefficient in T rather than in k, so
	// the whole series is one polynomial in T.
	return Horner(t, 2451550.09766, 29.530588861/lunationScale,
		0.00015437, -0.00000015, 0.00000000073)
}

// NewMoon returns the Julian ephemeris day of the New Moon nearest the given
// decimal year.
func NewMoon(year float64) float64 {
	return lunarPhase(lunationNumber(year, 0), &newMoonCoefficients)
}

// FullMoon returns the Julian ephemeris day of the Full Moon nearest the given
// decimal year.
func FullMoon(year float64) float64 {
	return lunarPhase(lunationNumber(year, .5), &fullMoonCoefficients)
}

// lunarPhase returns the Julian ephemeris day of the true phase at lunation
// index k, given the correction coefficients for New or Full Moon.
func lunarPhase(k float64, c *[25]float64) float64 {
	a := newLunarArguments(k)
	return meanLunarPhase(k) + a.correction(c) + a.additional()
}

// lunarArguments holds the arguments of the phase-correction series at one
// lunation index (Meeus, p. 350): the eccentricity factor, the Sun's and Moon's
// mean anomalies, the Moon's argument of latitude, the longitude of the
// ascending node, and the fourteen planetary arguments. Angles are radians.
type lunarArguments struct {
	e            float64
	m, mp, f, om float64
	planetary    [14]float64
}

// newLunarArguments evaluates the phase-correction arguments at lunation index
// k (Meeus, pp. 350 and 351).
func newLunarArguments(k float64) *lunarArguments {
	t := k * lunationScale
	a := &lunarArguments{
		e: Horner(t, 1, -0.002516, -0.0000074),
		m: radians(Horner(t, 2.5534, 29.1053567/lunationScale,
			-0.0000014, -0.00000011)),
		mp: radians(Horner(t, 201.5643, 385.81693528/lunationScale,
			0.0107582, 0.00001238, -0.000000058)),
		f: radians(Horner(t, 160.7108, 390.67050284/lunationScale,
			-0.0016118, -0.00000227, 0.000000011)),
		om: radians(Horner(t, 124.7746, -1.56375588/lunationScale,
			0.0020672, 0.00000215)),
	}
	// The fourteen planetary arguments, all in degrees before conversion.
	//
	// Fixed against the source this port derives from: the constant term of A1
	// is 299.77, not 299.7, and its quadratic term was left in degrees while
	// the rest of the expression had been converted to radians. A7 is 207.14,
	// not 207.17. All three values were confirmed against a second,
	// independently written rendering of the book.
	a.planetary = [14]float64{
		radians(299.77 + 0.107408*k - 0.009173*t*t),
		radians(251.88 + 0.016321*k),
		radians(251.83 + 26.651886*k),
		radians(349.42 + 36.412478*k),
		radians(84.66 + 18.206239*k),
		radians(141.74 + 53.303771*k),
		radians(207.14 + 2.453732*k),
		radians(154.84 + 7.306860*k),
		radians(34.52 + 27.261239*k),
		radians(207.19 + 0.121824*k),
		radians(291.34 + 1.844379*k),
		radians(161.72 + 24.198154*k),
		radians(239.56 + 25.513099*k),
		radians(331.55 + 3.592518*k),
	}
	return a
}

// correction returns the periodic correction to the mean phase, in days, for
// the given New or Full Moon coefficients (Meeus, p. 351). The twenty-five
// arguments are shared by both phases; only the amplitudes differ.
func (a *lunarArguments) correction(c *[25]float64) float64 {
	e, e2 := a.e, a.e*a.e
	m, mp, f, om := a.m, a.mp, a.f, a.om
	return c[0]*math.Sin(mp) +
		c[1]*math.Sin(m)*e +
		c[2]*math.Sin(2*mp) +
		c[3]*math.Sin(2*f) +
		c[4]*math.Sin(mp-m)*e +
		c[5]*math.Sin(mp+m)*e +
		c[6]*math.Sin(2*m)*e2 +
		c[7]*math.Sin(mp-2*f) +
		c[8]*math.Sin(mp+2*f) +
		c[9]*math.Sin(2*mp+m)*e +
		c[10]*math.Sin(3*mp) +
		c[11]*math.Sin(m+2*f)*e +
		c[12]*math.Sin(m-2*f)*e +
		c[13]*math.Sin(2*mp-m)*e +
		c[14]*math.Sin(om) +
		c[15]*math.Sin(mp+2*m) +
		c[16]*math.Sin(2*(mp-f)) +
		c[17]*math.Sin(3*m) +
		c[18]*math.Sin(mp+m-2*f) +
		c[19]*math.Sin(2*(mp+f)) +
		c[20]*math.Sin(mp+m+2*f) +
		c[21]*math.Sin(mp-m+2*f) +
		c[22]*math.Sin(mp-m-2*f) +
		c[23]*math.Sin(3*mp+m) +
		c[24]*math.Sin(4*mp)
}

// additional returns the fourteen further corrections, in days, that apply to
// every phase (Meeus, p. 352). They account for the perturbations of Venus and
// Jupiter and for the flattening of the Earth.
func (a *lunarArguments) additional() float64 {
	var sum float64
	for i, amplitude := range lunarPlanetaryAmplitudes {
		sum += amplitude * math.Sin(a.planetary[i])
	}
	return sum
}

// lunarPlanetaryAmplitudes are the amplitudes, in days, of the fourteen
// planetary corrections (Meeus, p. 352).
var lunarPlanetaryAmplitudes = [14]float64{
	0.000325, 0.000165, 0.000164, 0.000126, 0.000110, 0.000062, 0.000060,
	0.000056, 0.000047, 0.000042, 0.000040, 0.000037, 0.000035, 0.000023,
}

// newMoonCoefficients are the amplitudes, in days, of the twenty-five periodic
// corrections for New Moon (Meeus, p. 351).
var newMoonCoefficients = [25]float64{
	-0.40720, 0.17241, 0.01608, 0.01039, 0.00739,
	-0.00514, 0.00208, -0.00111, -0.00057, 0.00056,
	-0.00042, 0.00042, 0.00038, -0.00024, -0.00017,
	-0.00007, 0.00004, 0.00004, 0.00003, 0.00003,
	-0.00003, 0.00003, -0.00002, -0.00002, 0.00002,
}

// fullMoonCoefficients are the amplitudes, in days, of the twenty-five periodic
// corrections for Full Moon (Meeus, p. 351).
var fullMoonCoefficients = [25]float64{
	-0.40614, 0.17302, 0.01614, 0.01043, 0.00734,
	-0.00515, 0.00209, -0.00111, -0.00057, 0.00056,
	-0.00042, 0.00042, 0.00038, -0.00024, -0.00017,
	-0.00007, 0.00004, 0.00004, 0.00003, 0.00003,
	-0.00003, 0.00003, -0.00002, -0.00002, 0.00002,
}
