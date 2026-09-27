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

import (
	"math"
	"testing"
)

// TestApparentPlaceMeeus23a checks the apparent place against Meeus example
// 23.a: theta Persei on 2028 November 13.19 TD, from its J2000 place
// 2h44m11.986s +49 13 42.48 with the proper motion of example 21.b
// (+0.03425 s and -0.0895 arc second a year), comes to 2h46m14.390s
// +49 21 07.45. The book uses the same precession and nutation, and the
// Ron-Vondrak aberration where this uses the VSOP87 velocity; they agree to
// 0.02 arc second.
func TestApparentPlaceMeeus23a(t *testing.T) {
	t.Parallel()
	jde := 2462088.69
	years := (jde - 2451545) / 365.25
	ra0 := (2+44.0/60+11.986/3600)*15 + 0.03425*15/3600*years
	dec0 := 49 + 13.0/60 + 42.48/3600 - 0.0895/3600*years
	dpsi, deps := Nutation(jde)
	ra, dec := ApparentFromJ2000(ra0, dec0, jde, dpsi, deps, 0)
	wantRA := (2 + 46.0/60 + 14.390/3600) * 15
	wantDec := 49 + 21.0/60 + 7.45/3600
	dRA := (ra - wantRA) * math.Cos(radians(dec)) * 3600
	dDec := (dec - wantDec) * 3600
	if math.Hypot(dRA, dDec) > 0.05 {
		t.Errorf("theta Persei apparent place off Meeus by %.3f arcsec in RA and %.3f in Dec", dRA, dDec)
	}
}

// TestApparentPlaceParallax checks the annual parallax of the nearest star:
// Alpha Centauri at 1.34 pc moves by up to 0.75 arc second between the Sun
// and the Earth, and a star with no distance does not move at all.
func TestApparentPlaceParallax(t *testing.T) {
	t.Parallel()
	var worst float64
	for day := 0.0; day < 366; day += 7 {
		jde := 2461000.5 + day
		dpsi, deps := Nutation(jde)
		ra1, dec1 := ApparentFromJ2000(219.9, -60.8, jde, dpsi, deps, 1.3474)
		ra0, dec0 := ApparentFromJ2000(219.9, -60.8, jde, dpsi, deps, 0)
		d := math.Hypot((ra1-ra0)*math.Cos(radians(dec0)), dec1-dec0) * 3600
		worst = math.Max(worst, d)
	}
	if worst < 0.6 || worst > 0.76 {
		t.Errorf("Alpha Centauri parallax up to %.3f arcsec, want about 0.74", worst)
	}
}
