// Package elp evaluates the ELP 2000-82B lunar theory of M. Chapront-Touze
// and J. Chapront (Astronomy and Astrophysics 124, 50, 1983, and 190, 342,
// 1988), with its constants fitted to the JPL DE200/LE200 integration, as
// IMCCE distributes it with the Fortran subroutine ELP82B_2. The series is
// generated from IMCCE's 36 files by internal/cmd/genelp and truncated by
// amplitude; see the header of table.go.
//
// Meeus's chapter 47, which ephemeris.MoonPositionMeeus keeps, is an
// abridgement of the same theory to about 120 terms and 10 arc seconds; the
// table here keeps about a thousand and matches JPL DE441 to about half an
// arc second over the present era.
package elp

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

//go:generate go run ../cmd/genelp -out table.go

import "math"

// mainTerm is a main-problem term: a sin(p0 + p1 T + p2 T^2 + p3 T^3 + p4 T^4).
type mainTerm struct {
	a float64
	p [5]float64
}

// term is a perturbation term: a sin(p0 + p1 T), times T^k for the series it
// sits in.
type term struct {
	a, p0, p1 float64
}

// coordinate is the series of one coordinate.
type coordinate struct {
	main          []mainTerm
	perturbations [3][]term // multiplied by T^0, T^1, and T^2
}

const (
	j2000      = 2451545.0
	century    = 36525.0
	arcsecRad  = 648000 / math.Pi // arc seconds per radian
	distScale  = 384747.9806448954 / 384747.9806743165
	radToDeg   = 180 / math.Pi
	fullCircle = 2 * math.Pi
)

// Position returns the Moon's geocentric ecliptic longitude, in degrees in
// [0, 360), latitude in degrees, and distance in km, at the Julian ephemeris
// day jde (TDB, which is TT to the 2 ms that matter nowhere here), referred to
// the mean ecliptic and equinox of date, the frame of Meeus's chapter 47.
//
// ELP 2000-82B itself measures longitude on the ecliptic of date from the
// inertial departure point of J2000; Position adds the general precession in
// longitude (IAU 1976) to refer it to the mean equinox of date. Inertial
// returns the theory's own coordinates.
func Position(jde float64) (lonDeg, latDeg, distanceKm float64) {
	lon, lat, dist := Inertial(jde)
	t := (jde - j2000) / century
	lon += (5029.0966*t + 1.11113*t*t - 0.000006*t*t*t) / 3600
	lon = math.Mod(lon, 360)
	if lon < 0 {
		lon += 360
	}
	return lon, lat, dist
}

// Inertial returns the Moon's geocentric longitude, in degrees in [0, 360),
// latitude in degrees, and distance in km, at jde, in ELP 2000-82B's own
// frame: the mean ecliptic of date with longitude measured from the inertial
// departure point of J2000. ELP82B_2 rotates this to the ecliptic of J2000
// with Laskar's P and Q.
func Inertial(jde float64) (lonDeg, latDeg, distanceKm float64) {
	t := (jde - j2000) / century
	powers := [5]float64{1, t, t * t, t * t * t, t * t * t * t}
	var r [3]float64
	for c := range series {
		s := &series[c]
		var sum float64
		for i := range s.main {
			m := &s.main[i]
			sum += m.a * math.Sin(m.p[0]+m.p[1]*t+m.p[2]*powers[2]+m.p[3]*powers[3]+m.p[4]*powers[4])
		}
		for k := range s.perturbations {
			var part float64
			for i := range s.perturbations[k] {
				p := &s.perturbations[k][i]
				part += p.a * math.Sin(p.p0+p.p1*t)
			}
			sum += part * powers[k]
		}
		r[c] = sum
	}
	lon := r[0]/arcsecRad + meanLongitude[0] + meanLongitude[1]*t + meanLongitude[2]*powers[2] +
		meanLongitude[3]*powers[3] + meanLongitude[4]*powers[4]
	lon = math.Mod(lon, fullCircle)
	if lon < 0 {
		lon += fullCircle
	}
	return lon * radToDeg, r[1] / arcsecRad * radToDeg, r[2] * distScale
}
