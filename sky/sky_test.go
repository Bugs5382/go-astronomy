package sky_test

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
	"errors"
	"math"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/planet/mars"
	"github.com/Bugs5382/go-astronomy/planet/venus"
	"github.com/Bugs5382/go-astronomy/sky"
)

// TestEarthNeverRisesOrSetsFromTheNearSide checks the plain answer from the
// Moon: from the centre of the near side Earth stays above the horizon, and
// from the centre of the far side below it, for the whole search window, as
// the Horizons tables over the same two months confirm.
func TestEarthNeverRisesOrSetsFromTheNearSide(t *testing.T) {
	t.Parallel()
	from := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		file string
		lon  float64
		want sky.State
	}{
		{"earth-from-moon-0-0.txt", 0, sky.AlwaysAbove},
		{"earth-from-moon-180-0.txt", 180, sky.AlwaysBelow},
	} {
		for _, r := range load(t, c.file) {
			if above := r.num(t, 1) > 0; above != (c.want == sky.AlwaysAbove) {
				t.Fatalf("%s: Horizons has Earth at %.2f degrees at %s", c.file, r.num(t, 1), r.when)
			}
		}
		site := sky.Site{Body: sky.Moon, Lon: c.lon}
		rise, err := sky.NextRise(site, sky.Earth, from)
		if err != nil {
			t.Fatal(err)
		}
		set, err := sky.NextSet(site, sky.Earth, from)
		if err != nil {
			t.Fatal(err)
		}
		if rise.State != c.want || set.State != c.want || rise.Found() || !rise.Time.IsZero() {
			t.Errorf("lon %v: rise %+v, set %+v, want %s", c.lon, rise, set, c.want)
		}
	}
	if w := sky.Window(sky.Moon); w < 60*24*time.Hour || w > 70*24*time.Hour {
		t.Errorf("Moon window %v, want about 65 days", w)
	}
}

// TestSunFromTheMoon checks the lunar day: the Sun rises about once a
// synodic month, 29.53 days apart.
func TestSunFromTheMoon(t *testing.T) {
	t.Parallel()
	site := sky.Site{Body: sky.Moon}
	first, err := sky.NextRise(site, sky.Sun, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !first.Found() {
		t.Fatal(first, err)
	}
	second, err := sky.NextRise(site, sky.Sun, first.Time)
	if err != nil || !second.Found() {
		t.Fatal(second, err)
	}
	if d := second.Time.Sub(first.Time).Hours() / 24; math.Abs(d-29.53) > 0.5 {
		t.Errorf("sunrises %.2f days apart, want about 29.53", d)
	}
	if d := sky.Moon.SolarDay().Hours() / 24; math.Abs(d-29.53) > 0.01 {
		t.Errorf("Moon solar day %.3f days", d)
	}
}

// TestEarthPhaseFromTheMoon checks phases in the general case: Earth seen
// from the Moon is lit where the Moon seen from Earth is dark, so the two lit
// fractions add up to one (the phase angles are supplementary, less the
// Moon's parallax of the Sun).
func TestEarthPhaseFromTheMoon(t *testing.T) {
	t.Parallel()
	site := sky.Site{Body: sky.Moon}
	for d := 0; d < 30; d++ {
		when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
		res, err := sky.Position(site, sky.Earth, when)
		if err != nil {
			t.Fatal(err)
		}
		if sum := res.Illuminated + moon.Illumination(when); math.Abs(sum-1) > 0.005 {
			t.Errorf("%s: Earth %.4f lit from the Moon, Moon %.4f lit from Earth, sum %.4f", when.Format(time.DateOnly),
				res.Illuminated, moon.Illumination(when), sum)
		}
	}
}

// TestEarthSiteIsTheV1Answer checks the Earth case through the general form:
// a Site on Earth gives exactly the positions and events of the earth,
// earth/moon, and planet packages.
func TestEarthSiteIsTheV1Answer(t *testing.T) {
	t.Parallel()
	denver := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)}
	site := sky.Site{Body: sky.Earth, Lat: denver.Lat, Lon: denver.Lng, Height: denver.Height}
	marsBody := mustPlanet(t, mars.Planet)
	when := time.Date(2027, 6, 21, 3, 0, 0, 0, time.UTC)

	sun, err := sky.Position(site, sky.Sun, when)
	if err != nil || sun.Position != earth.SunPosition(denver, when) {
		t.Errorf("Sun %+v, v1 %+v (%v)", sun.Position, earth.SunPosition(denver, when), err)
	}
	mp, _ := moon.Position(denver, when)
	if got, err := sky.Position(site, sky.Moon, when); err != nil || got.Position != mp {
		t.Errorf("Moon %+v, v1 %+v (%v)", got.Position, mp, err)
	}
	pr, _ := mars.Position(denver, when)
	if got, err := sky.Position(site, marsBody, when); err != nil || got.Position != pr.Position || got.Illuminated != pr.Illuminated {
		t.Errorf("Mars %+v, v1 %+v (%v)", got, pr, err)
	}

	mr, _, _ := moon.NextRise(denver, when)
	if ev, err := sky.NextRise(site, sky.Moon, when); err != nil || !ev.Time.Equal(mr) {
		t.Errorf("moonrise %v, v1 %v (%v)", ev, mr, err)
	}
	ms, _, _ := mars.NextSet(denver, when)
	if ev, err := sky.NextSet(site, marsBody, when); err != nil || !ev.Time.Equal(ms) {
		t.Errorf("Mars set %v, v1 %v (%v)", ev, ms, err)
	}
	day, _ := earth.NewSunTimes(denver, when)
	var sunrise time.Time
	for _, s := range day.Segments() {
		if s.Label == earth.LabelSunrise {
			sunrise = s.From
			break
		}
	}
	if ev, err := sky.NextRise(site, sky.Sun, when); err != nil || !ev.Time.Equal(sunrise) {
		t.Errorf("sunrise %v, v1 %v (%v)", ev, sunrise, err)
	}
	noon, _ := day.SolarNoon()
	if ev, err := sky.NextTransit(site, sky.Sun, when); err != nil || !ev.Time.Equal(noon) {
		t.Errorf("solar noon %v, v1 %v (%v)", ev, noon, err)
	}

	segs, err := sky.Segments(site, day.DayStart(), day.DayEnd())
	if err != nil {
		t.Fatal(err)
	}
	v1 := day.Segments()
	if len(segs) != len(v1) {
		t.Fatalf("%d segments, v1 %d", len(segs), len(v1))
	}
	for i := range v1 {
		if segs[i].Label != v1[i].Label || !segs[i].From.Equal(v1[i].From) || !segs[i].To.Equal(v1[i].To) {
			t.Errorf("segment %d: %+v, v1 %+v", i, segs[i], v1[i])
		}
	}
}

// TestPolarSunOnEarth checks the midnight Sun and the polar night come back
// as states, not as a failure to find a sunrise.
func TestPolarSunOnEarth(t *testing.T) {
	t.Parallel()
	svalbard := sky.Site{Body: sky.Earth, Lat: 78.22, Lon: 15.65}
	for _, c := range []struct {
		when time.Time
		want sky.State
	}{
		{time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC), sky.AlwaysAbove},
		{time.Date(2027, 12, 21, 0, 0, 0, 0, time.UTC), sky.AlwaysBelow},
	} {
		ev, err := sky.NextRise(svalbard, sky.Sun, c.when)
		if err != nil || ev.State != c.want {
			t.Errorf("%s: %+v %v, want %s", c.when.Format(time.DateOnly), ev, err, c.want)
		}
	}
}

// TestAirlessDayHasNoTwilight checks an airless body's day divides at the
// Sun's horizon crossings and nowhere else: from the Moon, two months give
// alternating day and night spans of about two weeks, with no twilight
// label, split exactly at the sunrises and sunsets NextRise and NextSet find.
func TestAirlessDayHasNoTwilight(t *testing.T) {
	t.Parallel()
	site := sky.Site{Body: sky.Moon}
	if _, ok := sky.Moon.Twilight(); ok {
		t.Error("the Moon has a twilight")
	}
	if _, ok := sky.Earth.Twilight(); !ok {
		t.Error("the Earth has no twilight")
	}
	if sky.Moon.HorizonRefraction() != 0 {
		t.Errorf("Moon refraction %v, want 0", sky.Moon.HorizonRefraction())
	}
	from := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 2, 0)
	segs, err := sky.Segments(site, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) < 4 || !segs[0].From.Equal(from) || !segs[len(segs)-1].To.Equal(to) {
		t.Fatalf("segments %+v", segs)
	}
	for i, s := range segs {
		if s.Label != earth.LabelDay && s.Label != earth.LabelNight {
			t.Errorf("segment %d label %q", i, s.Label)
		}
		if i > 0 {
			if segs[i-1].Label == s.Label || !segs[i-1].To.Equal(s.From) {
				t.Errorf("segments %d and %d: %+v %+v", i-1, i, segs[i-1], s)
			}
			if i < len(segs)-1 && (s.To.Sub(s.From) < 13*24*time.Hour || s.To.Sub(s.From) > 17*24*time.Hour) {
				t.Errorf("segment %d lasts %v", i, s.To.Sub(s.From))
			}
			edge := sky.NextRise
			if s.Label == earth.LabelNight {
				edge = sky.NextSet
			}
			ev, err := edge(site, sky.Sun, s.From.Add(-time.Hour))
			if err != nil || ev.Time.Sub(s.From).Abs() > 2*time.Second {
				t.Errorf("segment %d starts %s, event %+v", i, s.From, ev)
			}
		}
	}
}

// TestSiteErrors checks the coded errors: no body, a site on the Sun, the
// site's own body as target, an out-of-range site, and a NaN height.
func TestSiteErrors(t *testing.T) {
	t.Parallel()
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		site   sky.Site
		target *sky.Body
		cause  error
		code   int
	}{
		{sky.Site{}, sky.Sun, sky.ErrUnknownBody, astronomy.CodeUnknownBody},
		{sky.Site{Body: sky.Moon}, nil, sky.ErrUnknownBody, astronomy.CodeUnknownBody},
		{sky.Site{Body: sky.Sun}, sky.Earth, sky.ErrUnknownBody, astronomy.CodeUnknownBody},
		{sky.Site{Body: sky.Moon}, sky.Moon, sky.ErrSameBody, astronomy.CodeSameBody},
		{sky.Site{Body: sky.Moon, Lat: 91}, sky.Earth, sky.ErrInvalidLatitude, astronomy.CodeInvalidLatitude},
		{sky.Site{Body: sky.Moon, Lon: -181}, sky.Earth, sky.ErrInvalidLongitude, astronomy.CodeInvalidLongitude},
		{sky.Site{Body: sky.Moon, Lat: math.NaN()}, sky.Earth, sky.ErrInvalidLatitude, astronomy.CodeInvalidLatitude},
		{sky.Site{Body: sky.Moon, Height: astronomy.Meters(math.Inf(1))}, sky.Earth, astronomy.ErrInvalidHeight, astronomy.CodeInvalidHeight},
	}
	for i, c := range cases {
		_, err := sky.Position(c.site, c.target, when)
		_, errRise := sky.NextRise(c.site, c.target, when)
		for _, e := range []error{err, errRise} {
			if code, _ := apperr.Code(e); !errors.Is(e, c.cause) || code != c.code {
				t.Errorf("case %d: %v, want %v with code %d", i, e, c.cause, c.code)
			}
		}
	}
	if _, err := sky.Planet(nil); !errors.Is(err, sky.ErrUnknownBody) {
		t.Errorf("Planet(nil) = %v", err)
	}
	if segs, err := sky.Segments(sky.Site{Body: sky.Moon}, when, when); err != nil || segs != nil {
		t.Errorf("empty window: %v %v", segs, err)
	}
}

// TestBodies checks the body descriptions: names, radii, and the rotation
// and solar day, including a retrograde spin.
func TestBodies(t *testing.T) {
	t.Parallel()
	v := mustPlanet(t, venus.Planet)
	m := mustPlanet(t, mars.Planet)
	cases := []struct {
		b        *sky.Body
		name     string
		solarDay time.Duration
		retro    bool
	}{
		{sky.Earth, "earth", 24 * time.Hour, false},
		{sky.Moon, "moon", time.Duration(29.5306 * 24 * float64(time.Hour)), false},
		{m, "mars", mars.Sol, false},
		{v, "venus", time.Duration(116.75 * 24 * float64(time.Hour)), true},
	}
	for _, c := range cases {
		if c.b.Name() != c.name {
			t.Errorf("name %q, want %q", c.b.Name(), c.name)
		}
		if d := c.b.SolarDay() - c.solarDay; d.Abs() > c.solarDay/2000 {
			t.Errorf("%s: solar day %v, want %v", c.name, c.b.SolarDay(), c.solarDay)
		}
		if (c.b.RotationPeriod() < 0) != c.retro {
			t.Errorf("%s: rotation %v", c.name, c.b.RotationPeriod())
		}
	}
	if m.RadiusKm() != mars.RadiusKm || m.PolarRadiusKm() >= m.RadiusKm() {
		t.Errorf("Mars radii %v %v", m.RadiusKm(), m.PolarRadiusKm())
	}
	if sky.Sun.SolarDay() != 0 {
		t.Errorf("Sun solar day %v", sky.Sun.SolarDay())
	}
	for _, s := range []sky.State{sky.Crossed, sky.AlwaysAbove, sky.AlwaysBelow, sky.NoEvent} {
		if s.String() == "" {
			t.Errorf("state %d has no name", s)
		}
	}
}
