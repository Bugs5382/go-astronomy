package moon_test

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// limbObserver is the observer of the Horizons bright-limb fixture.
var limbObserver = astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC}

// angleDiff returns the difference a - b folded into [-180, 180].
func angleDiff(a, b float64) float64 {
	d := math.Mod(a-b+540, 360) - 180
	return d
}

// bearing returns the position angle of the direction from point 2 toward
// point 1 on a sphere, in degrees in [0, 360), with longitude-like and
// latitude-like coordinates in degrees: measured from the pole of the
// latitude-like coordinate, turning toward increasing longitude. It is
// written out here, independently of the package, for the canvas check.
func bearing(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	y := math.Cos(lat1*r) * math.Sin((lon1-lon2)*r)
	x := math.Sin(lat1*r)*math.Cos(lat2*r) - math.Cos(lat1*r)*math.Sin(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Mod(math.Atan2(y, x)/r+360, 360)
}

// toJ2000 turns a position angle at the Moon, measured from the north pole of
// date, into one measured from the J2000 (ICRF) pole, which is the frame
// Horizons reports PsAng in. Precession since 2000 turns the pole by about 0.15
// degrees at the Moon by 2027. It steps a degree along the angle from the Moon,
// precesses both points to J2000, and measures the angle again.
func toJ2000(t *testing.T, when time.Time, paDeg float64) float64 {
	t.Helper()
	ra, dec := moonRADec(t, when)
	const step = 1.0 * math.Pi / 180
	d := dec * math.Pi / 180
	p := paDeg * math.Pi / 180
	dec2 := math.Asin(math.Sin(d)*math.Cos(step) + math.Cos(d)*math.Sin(step)*math.Cos(p))
	ra2 := ra + math.Atan2(math.Sin(p)*math.Sin(step)*math.Cos(d), math.Cos(step)-math.Sin(d)*math.Sin(dec2))*180/math.Pi
	year := 2000 + when.Sub(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)).Hours()/24/365.25
	a := coordinates.PrecessEquatorial(coordinates.Equatorial{RA: ra, Dec: dec}, year, 2000)
	b := coordinates.PrecessEquatorial(coordinates.Equatorial{RA: ra2, Dec: dec2 * 180 / math.Pi}, year, 2000)
	return coordinates.PositionAngle(b.RA, b.Dec, a.RA, a.Dec)
}

// moonRADec recovers the Moon's topocentric right ascension and declination
// of date from its horizontal position, for toJ2000.
func moonRADec(t *testing.T, when time.Time) (float64, float64) {
	t.Helper()
	m, err := moon.Position(limbObserver, when)
	if err != nil {
		t.Fatal(err)
	}
	eq := coordinates.HorizontalToEquatorial(coordinates.Horizontal{Altitude: m.Altitude, Azimuth: m.Azimuth},
		limbObserver.Lat, limbObserver.Lng, julian.ApparentSiderealTime(when))
	return eq.RA, eq.Dec
}

// TestBrightLimbAgainstHorizons checks the position angle of the bright limb
// against JPL Horizons DE441 (PsAng - 180) over a lunation for a topocentric
// observer. Horizons measures PsAng from the J2000 pole, so the library's angle
// of date is turned to that frame first. The observed worst is 0.007 degrees
// and the tolerance 0.05. Rows where less than 3% or more than 97% of the disc
// is lit are skipped: there the direction is ill-conditioned (issue 46).
func TestBrightLimbAgainstHorizons(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/horizons-bright-limb-2027-06.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	checked := 0
	worst := 0.0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		when, err := time.Parse(time.RFC3339, p[0])
		if err != nil {
			t.Fatal(err)
		}
		psang, _ := strconv.ParseFloat(p[1], 64)
		illu, _ := strconv.ParseFloat(p[2], 64)
		if illu < 3 || illu > 97 {
			continue
		}
		bl, err := moon.BrightLimbAt(limbObserver, when)
		if err != nil {
			t.Fatal(err)
		}
		want := math.Mod(psang+180, 360)
		d := math.Abs(angleDiff(toJ2000(t, when, bl.PositionAngle), want))
		worst = math.Max(worst, d)
		if d > 0.05 {
			t.Errorf("%s: position angle %.3f, Horizons %.3f (illuminated %.1f%%)", p[0], bl.PositionAngle, want, illu)
		}
		checked++
	}
	if checked < 25 {
		t.Fatalf("only %d rows checked", checked)
	}
	t.Logf("worst %.4f degrees over %d rows", worst, checked)
}

// TestBrightLimbFacesTheSunOnScreen is the canvas check: on the observer's sky,
// the bright limb points along the great circle from the Moon toward the Sun.
// The direction of the Sun from the Moon is computed here from the two
// horizontal positions alone, measured from the zenith direction and turning
// the same way as the position angle, and must match ZenithAngle.
func TestBrightLimbFacesTheSunOnScreen(t *testing.T) {
	t.Parallel()
	checked := 0
	for h := 0; h < 30*24; h += 5 {
		when := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(h) * time.Hour)
		m, err := moon.Position(limbObserver, when)
		if err != nil {
			t.Fatal(err)
		}
		if m.Altitude < 10 || m.Altitude > 60 {
			continue
		}
		if illu := moon.Illumination(when); illu < 0.03 || illu > 0.97 {
			continue
		}
		s := earth.SunPosition(limbObserver, when)
		// Horizontal frame: altitude plays declination, and azimuth runs the
		// other way round from right ascension as seen on the sky.
		want := bearing(-s.Azimuth, s.Altitude, -m.Azimuth, m.Altitude)
		bl, err := moon.BrightLimbAt(limbObserver, when)
		if err != nil {
			t.Fatal(err)
		}
		if d := math.Abs(angleDiff(bl.ZenithAngle, want)); d > 0.05 {
			t.Errorf("%s: ZenithAngle %.3f, direction to the Sun %.3f", when.Format(time.RFC3339), bl.ZenithAngle, want)
		}
		checked++
	}
	if checked < 10 {
		t.Fatalf("only %d instants checked", checked)
	}
}

// TestBrightLimbQuarters checks consistency with Illumination at the quarters:
// the terminator runs through the centre of the disc, and the bright limb
// faces within a degree of the Sun's direction, which is the elongation, a
// right angle, away from the anti-Sun side.
func TestBrightLimbQuarters(t *testing.T) {
	t.Parallel()
	for _, start := range []time.Time{
		time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC),
	} {
		// First quarter: the waxing Moon's bright limb faces west, toward the
		// evening Sun, so the position angle lies between 180 and 360; the
		// waning Moon's faces east, between 0 and 180.
		for h := 0; h < 30*24; h++ {
			when := start.Add(time.Duration(h) * time.Hour)
			illu := moon.Illumination(when)
			if math.Abs(illu-0.5) > 0.02 {
				continue
			}
			bl, err := moon.BrightLimbAt(limbObserver, when)
			if err != nil {
				t.Fatal(err)
			}
			waxing := moon.PhaseAt(when) <= moon.Full
			west := bl.PositionAngle > 180
			if waxing != west {
				t.Errorf("%s: half lit, waxing=%v, but position angle %.1f", when.Format(time.RFC3339), waxing, bl.PositionAngle)
			}
		}
	}
}

// TestBrightLimbRange checks every angle is reported in [0, 360), including
// right at new and full Moon, where the direction is defined but degenerate.
func TestBrightLimbRange(t *testing.T) {
	t.Parallel()
	for _, when := range []time.Time{
		moon.NextNew(time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)),
		moon.NextFull(time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)),
		time.Date(2027, 6, 10, 0, 0, 0, 0, time.UTC),
	} {
		bl, err := moon.BrightLimbAt(limbObserver, when)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range []float64{bl.PositionAngle, bl.ZenithAngle, bl.Parallactic + 180} {
			if !(a >= 0 && a < 360) || math.IsNaN(a) {
				t.Errorf("%s: angle %v out of range", when, a)
			}
		}
	}
	if _, err := moon.BrightLimbAt(astronomy.Observer{Lat: 91}, time.Now()); err == nil {
		t.Error("want an error for latitude 91")
	}
}
