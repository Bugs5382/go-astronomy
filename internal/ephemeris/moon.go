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

// Position of the Moon.
//
// Ported from Jean Meeus, Astronomical Algorithms, 2nd ed., chapter 47,
// "Position of the Moon": the fundamental arguments (47.1) through (47.5), the
// truncated ELP-2000/82 series of tables 47.A and 47.B, the additive terms of
// p. 342, and the equatorial horizontal parallax of p. 337. The Go form
// derives from soniakeys/meeus (moonposition), MIT licensed.
//
// The truncated series gives the geocentric longitude to about 10 arc seconds
// and the latitude to about 4 arc seconds. Both series tables were transcribed
// and then verified term by term against a second, independently written
// rendering of the book (PyMeeus) before use.

import "math"

// earthEquatorialRadiusKm is the Earth's equatorial radius, the baseline of the
// Moon's equatorial horizontal parallax (Meeus, p. 337).
const earthEquatorialRadiusKm = 6378.14

// MoonParallax returns the Moon's equatorial horizontal parallax, in degrees,
// at a distance of distanceKm between the centres of the Earth and the Moon
// (Meeus, p. 337).
func MoonParallax(distanceKm float64) float64 {
	return degrees(math.Asin(earthEquatorialRadiusKm / distanceKm))
}

// moonArguments returns the Moon's mean elongation, the Sun's mean anomaly, the
// Moon's mean anomaly, and the Moon's argument of latitude, all in radians, at
// T Julian centuries from J2000.0 (Meeus 47.2 through 47.5).
//
// Every rational coefficient is written with a decimal point. Writing 1/545868
// instead would be an untyped integer expression in Go, which evaluates to
// exactly zero and silently drops the cubic and quartic terms; that is the
// defect this port fixes.
func moonArguments(t float64) (d, m, mp, f float64) {
	d = radians(Horner(t, 297.8501921, 445267.1114034, -0.0018819, 1.0/545868, -1.0/113065000))
	// The quadratic coefficient of the Sun's mean anomaly is -0.0001536
	// (Meeus 47.3). The source this port derives from has -0.0001535.
	m = radians(Horner(t, 357.5291092, 35999.0502909, -0.0001536, 1.0/24490000))
	mp = radians(Horner(t, 134.9633964, 477198.8675055, 0.0087414, 1.0/69699, -1.0/14712000))
	f = radians(Horner(t, 93.2720950, 483202.0175233, -0.0036539, -1.0/3526000, 1.0/863310000))
	return
}

// MoonPosition returns the Moon's geocentric ecliptic longitude in degrees in
// [0, 360), its geocentric ecliptic latitude in degrees, and the distance
// between the centres of the Earth and the Moon in kilometres, at the given
// Julian ephemeris day.
//
// The longitude and latitude are referred to the mean equinox of date and do
// not include nutation. A caller wanting the apparent position adds the
// nutation in longitude; see Nutation.
func MoonPosition(jde float64) (lonDeg, latDeg, distanceKm float64) {
	t := J2000Century(jde)
	// Mean longitude of the Moon (47.1).
	lprime := Horner(t, 218.3164477, 481267.88123421, -0.0015786, 1.0/538841, -1.0/65194000)
	d, m, mp, f := moonArguments(t)
	lprimeRad := radians(pmod(lprime, 360))

	// Arguments of the additive terms (p. 342), due to the action of Venus,
	// Jupiter, and the flattening of the Earth.
	a1 := radians(119.75 + 131.849*t)
	a2 := radians(53.09 + 479264.290*t)
	a3 := radians(313.45 + 481266.484*t)

	// The eccentricity factor E scales the terms in the Sun's mean anomaly,
	// which the decreasing eccentricity of the Earth's orbit modulates (47.6).
	e := Horner(t, 1, -0.002516, -0.0000074)
	e2 := e * e

	sumL := 3958*math.Sin(a1) + 1962*math.Sin(lprimeRad-f) + 318*math.Sin(a2)
	sumR := 0.0
	for i := range moonLonDistTerms {
		term := &moonLonDistTerms[i]
		sin, cos := math.Sincos(term.d*d + term.m*m + term.mp*mp + term.f*f)
		scale := 1.0
		switch term.m {
		case 1, -1:
			scale = e
		case 2, -2:
			scale = e2
		}
		sumL += term.lon * sin * scale
		sumR += term.dist * cos * scale
	}

	sumB := -2235*math.Sin(lprimeRad) + 382*math.Sin(a3) +
		175*math.Sin(a1-f) + 175*math.Sin(a1+f) +
		127*math.Sin(lprimeRad-mp) - 115*math.Sin(lprimeRad+mp)
	for i := range moonLatTerms {
		term := &moonLatTerms[i]
		sin := math.Sin(term.d*d + term.m*m + term.mp*mp + term.f*f)
		scale := 1.0
		switch term.m {
		case 1, -1:
			scale = e
		case 2, -2:
			scale = e2
		}
		sumB += term.lat * sin * scale
	}

	// Table amplitudes are in units of 0.000001 degree for the longitude and
	// latitude series and 0.001 kilometre for the distance series.
	lonDeg = pmod(pmod(lprime, 360)+sumL*1e-6, 360)
	latDeg = sumB * 1e-6
	distanceKm = 385000.56 + sumR*1e-3
	return
}

// moonLonDistTerms is table 47.A: the periodic terms of the Moon's
// longitude and of the Earth-Moon distance. The first four columns are the
// integer multiples of the arguments D, M, M', and F; lon is in units of
// 0.000001 degree and dist in units of 0.001 kilometre.
var moonLonDistTerms = [...]struct{ d, m, mp, f, lon, dist float64 }{
	{0, 0, 1, 0, 6288774, -20905355},
	{2, 0, -1, 0, 1274027, -3699111},
	{2, 0, 0, 0, 658314, -2955968},
	{0, 0, 2, 0, 213618, -569925},
	{0, 1, 0, 0, -185116, 48888},
	{0, 0, 0, 2, -114332, -3149},
	{2, 0, -2, 0, 58793, 246158},
	{2, -1, -1, 0, 57066, -152138},
	{2, 0, 1, 0, 53322, -170733},
	{2, -1, 0, 0, 45758, -204586},
	{0, 1, -1, 0, -40923, -129620},
	{1, 0, 0, 0, -34720, 108743},
	{0, 1, 1, 0, -30383, 104755},
	{2, 0, 0, -2, 15327, 10321},
	{0, 0, 1, 2, -12528, 0},
	{0, 0, 1, -2, 10980, 79661},
	{4, 0, -1, 0, 10675, -34782},
	{0, 0, 3, 0, 10034, -23210},
	{4, 0, -2, 0, 8548, -21636},
	{2, 1, -1, 0, -7888, 24208},
	{2, 1, 0, 0, -6766, 30824},
	{1, 0, -1, 0, -5163, -8379},
	{1, 1, 0, 0, 4987, -16675},
	{2, -1, 1, 0, 4036, -12831},
	{2, 0, 2, 0, 3994, -10445},
	{4, 0, 0, 0, 3861, -11650},
	{2, 0, -3, 0, 3665, 14403},
	{0, 1, -2, 0, -2689, -7003},
	{2, 0, -1, 2, -2602, 0},
	{2, -1, -2, 0, 2390, 10056},
	{1, 0, 1, 0, -2348, 6322},
	{2, -2, 0, 0, 2236, -9884},
	{0, 1, 2, 0, -2120, 5751},
	{0, 2, 0, 0, -2069, 0},
	{2, -2, -1, 0, 2048, -4950},
	{2, 0, 1, -2, -1773, 4130},
	{2, 0, 0, 2, -1595, 0},
	{4, -1, -1, 0, 1215, -3958},
	{0, 0, 2, 2, -1110, 0},
	{3, 0, -1, 0, -892, 3258},
	{2, 1, 1, 0, -810, 2616},
	{4, -1, -2, 0, 759, -1897},
	{0, 2, -1, 0, -713, -2117},
	{2, 2, -1, 0, -700, 2354},
	{2, 1, -2, 0, 691, 0},
	{2, -1, 0, -2, 596, 0},
	{4, 0, 1, 0, 549, -1423},
	{0, 0, 4, 0, 537, -1117},
	{4, -1, 0, 0, 520, -1571},
	{1, 0, -2, 0, -487, -1739},
	{2, 1, 0, -2, -399, 0},
	{0, 0, 2, -2, -381, -4421},
	{1, 1, 1, 0, 351, 0},
	{3, 0, -2, 0, -340, 0},
	{4, 0, -3, 0, 330, 0},
	{2, -1, 2, 0, 327, 0},
	{0, 2, 1, 0, -323, 1165},
	{1, 1, -1, 0, 299, 0},
	{2, 0, 3, 0, 294, 0},
	{2, 0, -1, -2, 0, 8752},
}

// moonLatTerms is table 47.B: the periodic terms of the Moon's ecliptic
// latitude, in units of 0.000001 degree.
var moonLatTerms = [...]struct{ d, m, mp, f, lat float64 }{
	{0, 0, 0, 1, 5128122},
	{0, 0, 1, 1, 280602},
	{0, 0, 1, -1, 277693},
	{2, 0, 0, -1, 173237},
	{2, 0, -1, 1, 55413},
	{2, 0, -1, -1, 46271},
	{2, 0, 0, 1, 32573},
	{0, 0, 2, 1, 17198},
	{2, 0, 1, -1, 9266},
	{0, 0, 2, -1, 8822},
	{2, -1, 0, -1, 8216},
	{2, 0, -2, -1, 4324},
	{2, 0, 1, 1, 4200},
	{2, 1, 0, -1, -3359},
	{2, -1, -1, 1, 2463},
	{2, -1, 0, 1, 2211},
	{2, -1, -1, -1, 2065},
	{0, 1, -1, -1, -1870},
	{4, 0, -1, -1, 1828},
	{0, 1, 0, 1, -1794},
	{0, 0, 0, 3, -1749},
	{0, 1, -1, 1, -1565},
	{1, 0, 0, 1, -1491},
	{0, 1, 1, 1, -1475},
	{0, 1, 1, -1, -1410},
	{0, 1, 0, -1, -1344},
	{1, 0, 0, -1, -1335},
	{0, 0, 3, 1, 1107},
	{4, 0, 0, -1, 1021},
	{4, 0, -1, 1, 833},
	{0, 0, 1, -3, 777},
	{4, 0, -2, 1, 671},
	{2, 0, 0, -3, 607},
	{2, 0, 2, -1, 596},
	{2, -1, 1, -1, 491},
	{2, 0, -2, 1, -451},
	{0, 0, 3, -1, 439},
	{2, 0, 2, 1, 422},
	{2, 0, -3, -1, 421},
	{2, 1, -1, 1, -366},
	{2, 1, 0, 1, -351},
	{4, 0, 0, 1, 331},
	{2, -1, 1, 1, 315},
	{2, -2, 0, -1, 302},
	{0, 0, 1, 3, -283},
	{2, 1, 1, -1, -229},
	{1, 1, 0, -1, 223},
	{1, 1, 0, 1, 223},
	{0, 1, -2, -1, -220},
	{2, 1, -1, -1, -220},
	{1, 0, 1, 1, -185},
	{2, -1, -2, -1, 181},
	{0, 1, 2, 1, -177},
	{4, 0, -2, -1, 176},
	{4, -1, -1, -1, 166},
	{1, 0, 1, -1, -164},
	{4, 0, 1, -1, 132},
	{1, 0, -1, -1, -119},
	{4, -1, 0, -1, 115},
	{2, -2, 0, 1, 107},
}
