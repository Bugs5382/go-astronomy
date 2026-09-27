package satellite

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

// Frames and vectors: GMST 1982 for TEME, the WGS84 observer site and the
// geodetic sub-satellite point, and small vector helpers.

import (
	"math"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// siteECEF returns the observer's Earth-fixed position on the WGS84
// ellipsoid, in km.
func siteECEF(obs astronomy.Observer) [3]float64 {
	lat, lng := obs.Lat*math.Pi/180, obs.Lng*math.Pi/180
	e2 := wgs84F * (2 - wgs84F)
	sl := math.Sin(lat)
	n := wgs84A / math.Sqrt(1-e2*sl*sl)
	return [3]float64{
		n * math.Cos(lat) * math.Cos(lng),
		n * math.Cos(lat) * math.Sin(lng),
		n * (1 - e2) * sl,
	}
}

// geodetic returns the WGS84 geodetic latitude and longitude, in degrees,
// and height, in km, of an Earth-fixed position, by the usual fixed-point
// iteration on the latitude.
func geodetic(p [3]float64) (lat, lng, h float64) {
	e2 := wgs84F * (2 - wgs84F)
	lng = math.Atan2(p[1], p[0])
	rxy := math.Hypot(p[0], p[1])
	lat = math.Atan2(p[2], rxy*(1-e2))
	var n float64
	for range 6 {
		sl := math.Sin(lat)
		n = wgs84A / math.Sqrt(1-e2*sl*sl)
		lat = math.Atan2(p[2]+n*e2*sl, rxy)
	}
	h = rxy/math.Cos(lat) - n
	return lat * 180 / math.Pi, lng * 180 / math.Pi, h
}

func fromRADec(raDeg, decDeg, dist float64) [3]float64 {
	ra, dec := raDeg*math.Pi/180, decDeg*math.Pi/180
	return [3]float64{dist * math.Cos(dec) * math.Cos(ra), dist * math.Cos(dec) * math.Sin(ra), dist * math.Sin(dec)}
}

func sub(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func norm(a [3]float64) float64 { return math.Sqrt(dot(a, a)) }

func angleBetween(a, b [3]float64) float64 {
	c := dot(a, b) / (norm(a) * norm(b))
	return math.Acos(math.Max(-1, math.Min(1, c)))
}

// gmst82 returns Greenwich mean sidereal time in radians, 0 to 2 pi, from
// the IAU 1982 expression (Vallado 2004, eq. 3-45). SGP4 defines TEME
// against this angle, so it is the one to use when rotating TEME to an
// Earth-fixed frame.
func gmst82(jdUT1 float64) float64 {
	tut1 := (jdUT1 - 2451545.0) / 36525.0
	temp := -6.2e-6*tut1*tut1*tut1 + 0.093104*tut1*tut1 +
		(876600.0*3600+8640184.812866)*tut1 + 67310.54841 // seconds
	temp = math.Mod(temp*deg2rd/240.0, twoPi) // 360/86400 = 1/240, to degrees, to radians
	if temp < 0.0 {
		temp += twoPi
	}
	return temp
}
