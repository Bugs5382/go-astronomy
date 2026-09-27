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
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// horizonsSeries is a committed JPL Horizons Sun elevation series around solar
// midnight, with the lower culmination found from a finer series.
type horizonsSeries struct {
	culmination time.Time
	times       []time.Time
	elev        []float64
}

// loadHorizonsSeries reads one testdata/horizons-sun-*.txt fixture. The header
// of each file records the Horizons queries, the DE441 source, and the fetch
// date.
func loadHorizonsSeries(t *testing.T, name string) horizonsSeries {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()
	var s horizonsSeries
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if p[0] == "culmination" {
			p = p[1:]
			when, err := time.Parse(time.RFC3339, p[0])
			if err != nil {
				t.Fatalf("fixture %s: %v", name, err)
			}
			s.culmination = when
			continue
		}
		when, err := time.Parse(time.RFC3339, p[0])
		if err != nil {
			t.Fatalf("fixture %s: %v", name, err)
		}
		e, err := strconv.ParseFloat(p[1], 64)
		if err != nil {
			t.Fatalf("fixture %s: %v", name, err)
		}
		s.times = append(s.times, when)
		s.elev = append(s.elev, e)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if s.culmination.IsZero() || len(s.times) < 2 {
		t.Fatalf("fixture %s is incomplete", name)
	}
	return s
}

// crossing returns the instant, linearly interpolated between samples, at which
// the Horizons series crosses alt in the given direction.
func (s horizonsSeries) crossing(t *testing.T, alt float64, rising bool) time.Time {
	t.Helper()
	for i := 1; i < len(s.elev); i++ {
		a, b := s.elev[i-1]-alt, s.elev[i]-alt
		if (rising && a < 0 && b >= 0) || (!rising && a >= 0 && b < 0) {
			frac := a / (a - b)
			return s.times[i-1].Add(time.Duration(frac * float64(s.times[i].Sub(s.times[i-1]))))
		}
	}
	t.Fatalf("series never crosses %.2f (rising %v)", alt, rising)
	return time.Time{}
}

func within(a, b time.Time, tol time.Duration) bool {
	d := a.Sub(b)
	return d <= tol && d >= -tol
}

// seattle is the observer from issue 50.
func seattle(t *testing.T) astronomy.Observer {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	return astronomy.Observer{Lat: 47.61, Lng: -122.33, TZ: loc}
}

func segmentsOn(t *testing.T, obs astronomy.Observer, y int, m time.Month, d int) []earth.Segment {
	t.Helper()
	day, err := earth.NewSunTimes(obs, time.Date(y, m, d, 12, 0, 0, 0, obs.Location()))
	if err != nil {
		t.Fatalf("NewSunTimes: %v", err)
	}
	return day.Segments()
}

// TestDuskAcrossMidnightKeepsItsLabel is the reproduction from issue 50. In
// Seattle on 5 July 2027 the Sun is still sinking from -17.7 to -18 degrees
// between local midnight and 00:07 PDT (Horizons), so that span is the tail of
// astronomical dusk. It is followed by night and then a single astronomical
// dawn.
func TestDuskAcrossMidnightKeepsItsLabel(t *testing.T) {
	t.Parallel()
	obs := seattle(t)
	hz := loadHorizonsSeries(t, "horizons-sun-seattle-2027-07-05.txt")
	segs := segmentsOn(t, obs, 2027, time.July, 5)
	if len(segs) < 3 {
		t.Fatalf("got %d segments", len(segs))
	}
	want := []string{earth.LabelAstronomicalDusk, earth.LabelNight, earth.LabelAstronomicalDawn}
	for i, w := range want {
		if segs[i].Label != w {
			t.Errorf("segment %d = %s (%s to %s), want %s", i, segs[i].Label,
				segs[i].From.Format("15:04:05"), segs[i].To.Format("15:04:05"), w)
		}
	}
	if end := hz.crossing(t, -18, false); !within(segs[0].To, end, 30*time.Second) {
		t.Errorf("dusk ends %s, Horizons crosses -18 at %s", segs[0].To.UTC(), end)
	}

	prev := segmentsOn(t, obs, 2027, time.July, 4)
	if last := prev[len(prev)-1]; last.Label != earth.LabelAstronomicalDusk {
		t.Errorf("4 July ends with %s, want %s", last.Label, earth.LabelAstronomicalDusk)
	}
	// In this zone the night falls after midnight, so each date has one dawn.
	// Dusk legitimately appears twice: the tail of the previous evening at the
	// start of the date and the new evening at its end.
	for _, day := range [][]earth.Segment{prev, segs} {
		seen := map[string]bool{}
		for _, s := range day {
			if strings.HasSuffix(s.Label, "_dawn") {
				if seen[s.Label] {
					t.Errorf("%s appears twice on %s", s.Label, s.From.Format("Jan 02"))
				}
				seen[s.Label] = true
			}
		}
	}
}

// TestDawnBeforeMidnightKeepsItsLabel is the mirror case of issue 50: with the
// clock set far behind solar time the lower culmination falls before local
// midnight, and the Sun is already climbing through -18 degrees before the date
// ends. 4 July's last band is astronomical dawn.
func TestDawnBeforeMidnightKeepsItsLabel(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 47.61, Lng: -122.33, TZ: time.FixedZone("UTC-9:30", -(9*3600 + 1800))}
	hz := loadHorizonsSeries(t, "horizons-sun-seattle-2027-07-05.txt")
	segs := segmentsOn(t, obs, 2027, time.July, 4)
	last := segs[len(segs)-1]
	if last.Label != earth.LabelAstronomicalDawn {
		t.Fatalf("4 July ends with %s (%s to %s), want %s", last.Label,
			last.From.Format("15:04:05"), last.To.Format("15:04:05"), earth.LabelAstronomicalDawn)
	}
	if start := hz.crossing(t, -18, true); !within(last.From, start, 30*time.Second) {
		t.Errorf("dawn starts %s, Horizons crosses -18 at %s", last.From.UTC(), start)
	}
	next := segmentsOn(t, obs, 2027, time.July, 5)
	if next[0].Label != earth.LabelAstronomicalDawn {
		t.Errorf("5 July starts with %s, want %s", next[0].Label, earth.LabelAstronomicalDawn)
	}

	// SegmentAt stitches the band across the date line with one label.
	at := time.Date(2027, time.July, 4, 23, 58, 0, 0, obs.Location())
	band, _, err := earth.SegmentAt(obs, at)
	if err != nil {
		t.Fatalf("SegmentAt: %v", err)
	}
	if band.Label != earth.LabelAstronomicalDawn {
		t.Errorf("SegmentAt(%s) = %s, want %s", at, band.Label, earth.LabelAstronomicalDawn)
	}
}

// TestTwilightSplitsAtSolarMidnight checks a night with no true night: in
// Edinburgh in July the Sun bottoms out at -11.2 degrees, so nautical twilight
// runs all night. The band splits at the lower culmination, dusk before it and
// dawn after, within a minute of the Horizons instant.
func TestTwilightSplitsAtSolarMidnight(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	obs := astronomy.Observer{Lat: 55.95, Lng: -3.19, TZ: loc}
	hz := loadHorizonsSeries(t, "horizons-sun-edinburgh-2027-07-05.txt")
	segs := segmentsOn(t, obs, 2027, time.July, 5)
	found := false
	for i := 1; i < len(segs); i++ {
		if segs[i-1].Label == earth.LabelNauticalDusk && segs[i].Label == earth.LabelNauticalDawn {
			found = true
			if !within(segs[i].From, hz.culmination, time.Minute) {
				t.Errorf("split at %s, Horizons lower culmination at %s", segs[i].From.UTC(), hz.culmination)
			}
		}
	}
	if !found {
		for _, s := range segs {
			t.Logf("%s %s to %s", s.Label, s.From.Format("15:04:05"), s.To.Format("15:04:05"))
		}
		t.Fatal("no nautical_dusk to nautical_dawn split")
	}
}

// TestTwilightLabelsFollowTheTrend is the invariant behind issue 50: along
// every date's segment list, a dawn-side label always has the Sun climbing and
// a dusk-side label always has it sinking. It samples the year at three
// latitudes where twilight crosses midnight for part of the year.
func TestTwilightLabelsFollowTheTrend(t *testing.T) {
	t.Parallel()
	rising := map[string]bool{
		earth.LabelAstronomicalDawn: true, earth.LabelNauticalDawn: true,
		earth.LabelCivilDawn: true, earth.LabelSunrise: true,
	}
	setting := map[string]bool{
		earth.LabelAstronomicalDusk: true, earth.LabelNauticalDusk: true,
		earth.LabelCivilDusk: true, earth.LabelSunset: true,
	}
	sites := []astronomy.Observer{
		{Lat: 47.6, Lng: -122.33, TZ: time.FixedZone("PST", -8*3600)},
		{Lat: 60, Lng: 10.75, TZ: time.FixedZone("CET", 3600)},
		{Lat: -54.8, Lng: -68.3, TZ: time.FixedZone("ART", -3*3600)},
	}
	step := 5
	if testing.Short() {
		step = 30
	}
	for _, obs := range sites {
		for d := 0; d < 366; d += step {
			date := time.Date(2027, time.January, 1+d, 12, 0, 0, 0, obs.Location())
			day, err := earth.NewSunTimes(obs, date)
			if err != nil {
				t.Fatalf("NewSunTimes: %v", err)
			}
			for _, s := range day.Segments() {
				mid := s.From.Add(s.To.Sub(s.From) / 2)
				before := earth.SunPosition(obs, mid.Add(-30*time.Second)).Altitude
				after := earth.SunPosition(obs, mid.Add(30*time.Second)).Altitude
				if rising[s.Label] && after <= before {
					t.Errorf("%.1f on %s: %s %s to %s while the Sun sinks", obs.Lat,
						date.Format("2006-01-02"), s.Label, s.From.Format("15:04"), s.To.Format("15:04"))
				}
				if setting[s.Label] && after >= before {
					t.Errorf("%.1f on %s: %s %s to %s while the Sun climbs", obs.Lat,
						date.Format("2006-01-02"), s.Label, s.From.Format("15:04"), s.To.Format("15:04"))
				}
			}
		}
	}
}
