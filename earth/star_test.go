package earth_test

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
	"time"

	meeuscoord "github.com/soniakeys/meeus/v3/coord"
	meeusjulian "github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/precess"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/unit"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/star"
)

// starPipeline reproduces the exact wiring earth.StarPosition is expected to
// use: precess the J2000 catalog position to the epoch of date, then the meeus
// horizontal transform with west-positive longitude and a 180 degree azimuth
// rotation to clockwise-from-north.
func starPipeline(raJ2000, decJ2000 float64, t time.Time, latDeg, lngEastDeg float64) (alt, az float64) {
	jd := meeusjulian.TimeToJD(t.UTC())
	epoch := 2000.0 + (jd-2451545.0)/365.25
	from := &meeuscoord.Equatorial{RA: unit.RAFromDeg(raJ2000), Dec: unit.AngleFromDeg(decJ2000)}
	to := &meeuscoord.Equatorial{}
	precess.Position(from, to, 2000.0, epoch, 0, 0)
	gst := sidereal.Mean(jd).Hour() * 15
	st := unit.TimeFromHour(gst / 15)
	a, h := meeuscoord.EqToHz(to.RA, to.Dec, unit.AngleFromDeg(latDeg), unit.AngleFromDeg(-lngEastDeg), st)
	az = math.Mod(a.Deg()+180+360, 360)
	alt = h.Deg()
	return
}

// expectedStarAltitude independently computes a star's geometric altitude with
// plain spherical trigonometry on the precessed equatorial position and mean
// sidereal time, a cross-check of the horizontal transform rather than a copy.
func expectedStarAltitude(raJ2000, decJ2000 float64, t time.Time, latDeg, lngEastDeg float64) float64 {
	jd := meeusjulian.TimeToJD(t.UTC())
	epoch := 2000.0 + (jd-2451545.0)/365.25
	from := &meeuscoord.Equatorial{RA: unit.RAFromDeg(raJ2000), Dec: unit.AngleFromDeg(decJ2000)}
	to := &meeuscoord.Equatorial{}
	precess.Position(from, to, 2000.0, epoch, 0, 0)
	gst := sidereal.Mean(jd).Hour() * 15
	hourAngle := (gst + lngEastDeg - to.RA.Deg()) * deg2rad
	latR := latDeg * deg2rad
	decR := to.Dec.Rad()
	sinAlt := math.Sin(latR)*math.Sin(decR) + math.Cos(latR)*math.Cos(decR)*math.Cos(hourAngle)
	return math.Asin(sinAlt) / deg2rad
}

func mustStar(t *testing.T, name string) star.Star {
	t.Helper()
	s, ok := star.GetNamedStar(name)
	if !ok {
		t.Fatalf("star %q not in catalog", name)
	}
	return s
}

func TestStarPositionMatchesPipeline(t *testing.T) {
	t.Parallel()
	stars := []star.Star{
		mustStar(t, "Sirius"),
		mustStar(t, "Vega"),
		mustStar(t, "Betelgeuse"),
		mustStar(t, "Polaris"),
	}
	instants := []time.Time{
		time.Date(2026, 1, 15, 3, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 4, 22, 30, 0, 0, time.UTC),
		time.Date(1982, 5, 3, 6, 0, 0, 0, time.UTC),
	}
	observers := []astronomy.Observer{brooklyn, quito, tromso, auckland}
	for _, s := range stars {
		for _, obs := range observers {
			for _, when := range instants {
				hz, err := earth.StarPosition(s, obs, when)
				if err != nil {
					t.Fatalf("StarPosition(%s) error: %v", s.ProperName, err)
				}
				wantAlt, wantAz := starPipeline(s.RA, s.Dec, when, obs.Lat, obs.Lng)
				if math.Abs(hz.Altitude-wantAlt) > 1e-9 {
					t.Errorf("%s %v alt = %.12f, want %.12f", s.ProperName, when, hz.Altitude, wantAlt)
				}
				if math.Abs(hz.Azimuth-wantAz) > 1e-9 {
					t.Errorf("%s %v az = %.12f, want %.12f", s.ProperName, when, hz.Azimuth, wantAz)
				}
				if hz.Azimuth < 0 || hz.Azimuth >= 360 {
					t.Errorf("%s %v azimuth %v out of [0,360)", s.ProperName, when, hz.Azimuth)
				}
			}
		}
	}
}

func TestStarPositionAltitudeIndependent(t *testing.T) {
	t.Parallel()
	sirius := mustStar(t, "Sirius")
	vega := mustStar(t, "Vega")
	when := time.Date(2026, 2, 10, 2, 0, 0, 0, time.UTC)
	for _, s := range []star.Star{sirius, vega} {
		for _, obs := range []astronomy.Observer{brooklyn, quito, auckland} {
			hz, err := earth.StarPosition(s, obs, when)
			if err != nil {
				t.Fatal(err)
			}
			want := expectedStarAltitude(s.RA, s.Dec, when, obs.Lat, obs.Lng)
			if math.Abs(hz.Altitude-want) > 1e-4 {
				t.Errorf("%s @ %v altitude = %.6f, want %.6f", s.ProperName, obs, hz.Altitude, want)
			}
		}
	}
}

// TestStarPositionPolarisAltitude verifies the classic identity that Polaris
// sits at an altitude close to the observer's latitude in the northern
// hemisphere, at any time of night. Polaris is about 0.7 degrees from the
// celestial pole, so the altitude tracks latitude to within roughly that much.
func TestStarPositionPolarisAltitude(t *testing.T) {
	t.Parallel()
	polaris := mustStar(t, "Polaris")
	observers := []astronomy.Observer{brooklyn, tromso}
	hours := []int{0, 6, 12, 18}
	for _, obs := range observers {
		for _, h := range hours {
			when := time.Date(2026, 9, 5, h, 0, 0, 0, time.UTC)
			hz, err := earth.StarPosition(polaris, obs, when)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(hz.Altitude-obs.Lat) > 1.0 {
				t.Errorf("Polaris altitude %.3f at lat %.3f (%dh), want within 1 degree",
					hz.Altitude, obs.Lat, h)
			}
		}
	}
}

func TestStarPositionInvalidObserver(t *testing.T) {
	t.Parallel()
	sirius := mustStar(t, "Sirius")
	when := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		obs  astronomy.Observer
		code int
	}{
		{astronomy.Observer{Lat: 120, Lng: 0}, astronomy.CodeInvalidLatitude},
		{astronomy.Observer{Lat: -91, Lng: 0}, astronomy.CodeInvalidLatitude},
		{astronomy.Observer{Lat: 0, Lng: 200}, astronomy.CodeInvalidLongitude},
		{astronomy.Observer{Lat: 0, Lng: -181}, astronomy.CodeInvalidLongitude},
	}
	for _, c := range cases {
		_, err := earth.StarPosition(sirius, c.obs, when)
		if err == nil {
			t.Errorf("StarPosition(%v) expected error", c.obs)
			continue
		}
		if got, ok := apperr.Code(err); !ok || got != c.code {
			t.Errorf("StarPosition(%v) code = (%d,%v), want %d", c.obs, got, ok, c.code)
		}
	}
}

// TestStarPositionAboveHorizon confirms the AboveHorizon helper agrees with the
// sign of the altitude across a full day for a star and observer.
func TestStarPositionAboveHorizon(t *testing.T) {
	t.Parallel()
	sirius := mustStar(t, "Sirius")
	start := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	var up, down int
	for i := 0; i < 24; i++ {
		when := start.Add(time.Duration(i) * time.Hour)
		hz, err := earth.StarPosition(sirius, brooklyn, when)
		if err != nil {
			t.Fatal(err)
		}
		if hz.AboveHorizon() != (hz.Altitude > 0) {
			t.Errorf("AboveHorizon %v disagrees with altitude %.3f", hz.AboveHorizon(), hz.Altitude)
		}
		if hz.AboveHorizon() {
			up++
		} else {
			down++
		}
	}
	// Sirius rises and sets from Brooklyn, so both states occur within a day.
	if up == 0 || down == 0 {
		t.Errorf("Sirius never changed horizon state: up=%d down=%d", up, down)
	}
}
