package elp_test

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
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Bugs5382/go-astronomy/internal/elp"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

// toJ2000 is ELP82B_2's closing rotation, Laskar's P and Q: from the mean
// ecliptic and equinox of date to the inertial ecliptic of J2000, the frame
// JPL Horizons gives the vectors in.
func toJ2000(lonDeg, latDeg, distKm, jde float64) [3]float64 {
	t := (jde - 2451545) / 36525
	lon, lat := lonDeg*math.Pi/180, latDeg*math.Pi/180
	x1 := distKm * math.Cos(lat)
	x2 := x1 * math.Sin(lon)
	x1 *= math.Cos(lon)
	x3 := distKm * math.Sin(lat)
	pw := (0.10180391e-4 + (0.47020439e-6+(-0.5417367e-9+(-0.2507948e-11+0.463486e-14*t)*t)*t)*t) * t
	qw := (-0.113469002e-3 + (0.12372674e-6+(0.1265417e-8+(-0.1371808e-11-0.320334e-14*t)*t)*t)*t) * t
	ra := 2 * math.Sqrt(1-pw*pw-qw*qw)
	pwqw := 2 * pw * qw
	pw2 := 1 - 2*pw*pw
	qw2 := 1 - 2*qw*qw
	pw *= ra
	qw *= ra
	return [3]float64{
		pw2*x1 + pwqw*x2 + pw*x3,
		pwqw*x1 + qw2*x2 - qw*x3,
		-pw*x1 + qw*x2 + (pw2+qw2-1)*x3,
	}
}

// precession is the general precession in longitude from J2000, in degrees
// (IAU 1976), which takes a longitude from the mean equinox of date back to
// the inertial departure point.
func precession(jde float64) float64 {
	t := (jde - 2451545) / 36525
	return (5029.0966*t + 1.11113*t*t - 0.000006*t*t*t) / 3600
}

// TestPositionIsInertialPlusPrecession checks Position is Inertial referred
// to the mean equinox of date, and agrees with Meeus chapter 47, which is in
// that frame, to its 10 arc seconds.
func TestPositionIsInertialPlusPrecession(t *testing.T) {
	t.Parallel()
	for jde := 2433282.5; jde < 2488070; jde += 997.3 {
		lon, lat, dist := elp.Position(jde)
		ilon, ilat, idist := elp.Inertial(jde)
		if d := math.Mod(lon-ilon-precession(jde)+540, 360) - 180; math.Abs(d) > 1e-9 || lat != ilat || dist != idist {
			t.Fatalf("jde %v: Position %v %v %v, Inertial %v %v %v", jde, lon, lat, dist, ilon, ilat, idist)
		}
		mlon, mlat, mdist := ephemeris.MoonPositionMeeus(jde)
		dlon := (math.Mod(lon-mlon+540, 360) - 180) * 3600
		if math.Abs(dlon) > 20 || math.Abs(lat-mlat)*3600 > 10 || math.Abs(dist-mdist) > 30 {
			t.Errorf("jde %v: ELP and Meeus 47 differ by %.1f arcsec, %.1f arcsec, %.1f km", jde, dlon, (lat-mlat)*3600, dist-mdist)
		}
	}
}

type vector struct {
	jde float64
	r   [3]float64
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	f, err := os.Open("testdata/horizons-moon-geocentric.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []vector
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		p := strings.Fields(line)
		var v vector
		v.jde, _ = strconv.ParseFloat(p[0], 64)
		for i := range 3 {
			v.r[i], _ = strconv.ParseFloat(p[1+i], 64)
		}
		out = append(out, v)
	}
	if len(out) < 150 {
		t.Fatalf("only %d rows", len(out))
	}
	return out
}

func angleArcsec(a, b [3]float64) float64 {
	dot := (a[0]*b[0] + a[1]*b[1] + a[2]*b[2]) / (norm(a) * norm(b))
	return math.Acos(math.Min(1, dot)) * 180 / math.Pi * 3600
}

func norm(a [3]float64) float64 { return math.Sqrt(a[0]*a[0] + a[1]*a[1] + a[2]*a[2]) }

// TestAgainstDE441 checks the truncated series against JPL Horizons DE441
// geocentric vectors over 1950 to 1955, 2024 to 2028, and 2095 to 2100, and
// against Meeus's chapter 47 over the same epochs. The direction agrees to
// 0.3 arc second in the 1950s and 0.55 in the 2020s (the full series is 0.35
// there, the fit of its constants to DE200), and drifts to 1.8 by the 2090s
// with the theory's secular terms; chapter 47 is off by 6.6, 9.0, and 9.8.
func TestAgainstDE441(t *testing.T) {
	t.Parallel()
	type worst struct{ elp, meeus, distKm float64 }
	spans := map[string]*worst{"1950s": {}, "2020s": {}, "2090s": {}}
	for _, v := range loadVectors(t) {
		span := "2020s"
		switch {
		case v.jde < 2440000:
			span = "1950s"
		case v.jde > 2480000:
			span = "2090s"
		}
		w := spans[span]
		lon, lat, dist := elp.Inertial(v.jde)
		got := toJ2000(lon, lat, dist, v.jde)
		w.elp = math.Max(w.elp, angleArcsec(got, v.r))
		w.distKm = math.Max(w.distKm, math.Abs(norm(got)-norm(v.r)))
		mlon, mlat, mdist := ephemeris.MoonPositionMeeus(v.jde)
		w.meeus = math.Max(w.meeus, angleArcsec(toJ2000(mlon-precession(v.jde), mlat, mdist, v.jde), v.r))
	}
	bounds := map[string]float64{"1950s": 0.5, "2020s": 0.6, "2090s": 2.0}
	for span, w := range spans {
		t.Logf("%s: ELP %.3f arcsec and %.3f km, Meeus 47 %.2f arcsec", span, w.elp, w.distKm, w.meeus)
		if w.elp > bounds[span] || w.distKm > 1 {
			t.Errorf("%s: ELP off DE441 by %.3f arcsec (want %.1f) and %.3f km (want 1)", span, w.elp, bounds[span], w.distKm)
		}
		if w.elp*5 > w.meeus {
			t.Errorf("%s: ELP %.3f arcsec is not five times better than Meeus 47's %.2f", span, w.elp, w.meeus)
		}
	}
}
