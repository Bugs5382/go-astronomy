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
	"errors"
	"math"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/sun"
)

// canonical is the design's reference date; only its calendar day is used, the
// civil day being resolved in each observer's own zone.
var canonical = time.Date(1982, 5, 3, 12, 0, 0, 0, time.UTC)

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Observer fixtures spanning hemispheres, longitudes, and latitudes.
var (
	brooklyn    = astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: mustLoad("America/New_York")}
	southampton = astronomy.Observer{Lat: 40.884, Lng: -72.390, TZ: mustLoad("America/New_York")}
	tromso      = astronomy.Observer{Lat: 69.6492, Lng: 18.9553, TZ: mustLoad("Europe/Oslo")}
	svalbard    = astronomy.Observer{Lat: 78.2232, Lng: 15.6267, TZ: mustLoad("Arctic/Longyearbyen")}
	ushuaia     = astronomy.Observer{Lat: -54.8019, Lng: -68.303, TZ: mustLoad("America/Argentina/Ushuaia")}
	quito       = astronomy.Observer{Lat: -0.1807, Lng: -78.4678, TZ: mustLoad("America/Guayaquil")}
	singapore   = astronomy.Observer{Lat: 1.3521, Lng: 103.8198, TZ: mustLoad("Asia/Singapore")}
	auckland    = astronomy.Observer{Lat: -36.8485, Lng: 174.7633, TZ: mustLoad("Pacific/Auckland")}
)

var allObservers = []struct {
	name string
	obs  astronomy.Observer
}{
	{"brooklyn", brooklyn},
	{"southampton", southampton},
	{"tromso", tromso},
	{"svalbard", svalbard},
	{"ushuaia", ushuaia},
	{"quito", quito},
	{"singapore", singapore},
	{"auckland", auckland},
}

// segmentByLabel returns the first segment carrying the given label and whether
// one was found.
func segmentByLabel(segs []earth.Segment, label string) (earth.Segment, bool) {
	for _, s := range segs {
		if s.Label == label {
			return s, true
		}
	}
	return earth.Segment{}, false
}

// clockDuration returns the wall-clock span between two instants in the same
// zone as a duration, tolerant of the different absolute days.
func clockDuration(from, to time.Time) time.Duration { return to.Sub(from) }

func TestRefraction(t *testing.T) {
	t.Parallel()
	// At the horizon the standard atmosphere lifts a body by roughly 34 arc
	// minutes, about 0.57 degrees.
	if got := earth.Refraction(0); math.Abs(got-0.57) > 0.02 {
		t.Errorf("Refraction(0) = %.5f deg, want ~0.57", got)
	}
	// Refraction falls monotonically from the horizon to the zenith.
	prev := math.Inf(1)
	for h := 0.0; h <= 90.0; h += 5 {
		r := earth.Refraction(h)
		if r > prev {
			t.Errorf("Refraction not monotonic: Refraction(%.0f)=%.5f > previous %.5f", h, r, prev)
		}
		prev = r
	}
	// Near the zenith the correction is negligible.
	if got := earth.Refraction(90); math.Abs(got) > 0.001 {
		t.Errorf("Refraction(90) = %.5f deg, want ~0", got)
	}
}

func TestDefaultSegmentationOrderedAndUsable(t *testing.T) {
	t.Parallel()
	levels := earth.DefaultSegmentation.Levels
	if len(levels) == 0 {
		t.Fatal("DefaultSegmentation has no levels")
	}
	for i := 1; i < len(levels); i++ {
		if levels[i].Altitude <= levels[i-1].Altitude {
			t.Errorf("levels not strictly ascending at %d: %.3f then %.3f",
				i, levels[i-1].Altitude, levels[i].Altitude)
		}
	}
	if earth.DefaultSegmentation.Horizon != earth.HorizonAltitude {
		t.Errorf("Horizon = %.4f, want %.4f", earth.DefaultSegmentation.Horizon, earth.HorizonAltitude)
	}
}

func TestSunTimesContiguousAndSumsToDay(t *testing.T) {
	t.Parallel()
	for _, f := range allObservers {
		st, err := earth.NewSunTimes(f.obs, canonical)
		if err != nil {
			t.Fatalf("%s: NewSunTimes: %v", f.name, err)
		}
		segs := st.Segments()
		if len(segs) == 0 {
			t.Fatalf("%s: no segments", f.name)
		}
		if !segs[0].From.Equal(st.DayStart()) {
			t.Errorf("%s: first segment starts %v, want day start %v", f.name, segs[0].From, st.DayStart())
		}
		if !segs[len(segs)-1].To.Equal(st.DayEnd()) {
			t.Errorf("%s: last segment ends %v, want day end %v", f.name, segs[len(segs)-1].To, st.DayEnd())
		}
		var sum float64
		for i, s := range segs {
			if !s.To.After(s.From) {
				t.Errorf("%s: segment %d not forward: %v -> %v", f.name, i, s.From, s.To)
			}
			if i > 0 && !s.From.Equal(segs[i-1].To) {
				t.Errorf("%s: gap at segment %d: %v != previous end %v", f.name, i, s.From, segs[i-1].To)
			}
			if math.Abs(s.Seconds-s.To.Sub(s.From).Seconds()) > 1e-6 {
				t.Errorf("%s: segment %d Seconds=%.3f mismatch span", f.name, i, s.Seconds)
			}
			sum += s.Seconds
		}
		want := st.DayLength().Seconds()
		if math.Abs(sum-want) > 1.0 {
			t.Errorf("%s: segments sum to %.1fs, want day length %.1fs", f.name, sum, want)
		}
	}
}

func TestSunriseSunsetKnownValues(t *testing.T) {
	t.Parallel()
	// Reference clock values for 1982-05-03 in local (EDT) time, consistent with
	// USNO tables to within a couple of minutes.
	cases := []struct {
		name         string
		obs          astronomy.Observer
		riseH, riseM int
		setH, setM   int
		tol          time.Duration
	}{
		{"brooklyn", brooklyn, 5, 52, 19, 53, 3 * time.Minute},
		{"southampton", southampton, 5, 45, 19, 47, 3 * time.Minute},
	}
	for _, c := range cases {
		st, err := earth.NewSunTimes(c.obs, canonical)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		segs := st.Segments()
		sunrise, ok := segmentByLabel(segs, earth.LabelSunrise)
		if !ok {
			t.Fatalf("%s: no sunrise band", c.name)
		}
		sunset, ok := segmentByLabel(segs, earth.LabelSunset)
		if !ok {
			t.Fatalf("%s: no sunset band", c.name)
		}
		loc := c.obs.Location()
		wantRise := time.Date(1982, 5, 3, c.riseH, c.riseM, 0, 0, loc)
		wantSet := time.Date(1982, 5, 3, c.setH, c.setM, 0, 0, loc)
		if d := sunrise.From.Sub(wantRise); d < -c.tol || d > c.tol {
			t.Errorf("%s: sunrise %v, want ~%v (off by %v)", c.name, sunrise.From, wantRise, d)
		}
		if d := sunset.To.Sub(wantSet); d < -c.tol || d > c.tol {
			t.Errorf("%s: sunset %v, want ~%v (off by %v)", c.name, sunset.To, wantSet, d)
		}
		noon, _ := st.SolarNoon()
		if !sunrise.From.Before(noon) || !noon.Before(sunset.To) {
			t.Errorf("%s: expected sunrise < noon < sunset, got %v %v %v", c.name, sunrise.From, noon, sunset.To)
		}
	}
}

// TestLongitudePrecision confirms the close pair resolves distinctly: the more
// easterly Southampton sees the Sun rise and set earlier on the clock than
// Brooklyn on the same day and zone.
func TestLongitudePrecision(t *testing.T) {
	t.Parallel()
	bk, err := earth.NewSunTimes(brooklyn, canonical)
	if err != nil {
		t.Fatal(err)
	}
	sh, err := earth.NewSunTimes(southampton, canonical)
	if err != nil {
		t.Fatal(err)
	}
	bkRise, _ := segmentByLabel(bk.Segments(), earth.LabelSunrise)
	shRise, _ := segmentByLabel(sh.Segments(), earth.LabelSunrise)
	bkSet, _ := segmentByLabel(bk.Segments(), earth.LabelSunset)
	shSet, _ := segmentByLabel(sh.Segments(), earth.LabelSunset)
	if !shRise.From.Before(bkRise.From) {
		t.Errorf("Southampton sunrise %v not before Brooklyn %v", shRise.From, bkRise.From)
	}
	if !shSet.To.Before(bkSet.To) {
		t.Errorf("Southampton sunset %v not before Brooklyn %v", shSet.To, bkSet.To)
	}
	// The two locations are close, so the offsets are small (a few minutes).
	if d := bkRise.From.Sub(shRise.From); d > 15*time.Minute {
		t.Errorf("sunrise offset %v larger than expected for a close pair", d)
	}
}

func TestSolarNoonIsAltitudePeak(t *testing.T) {
	t.Parallel()
	for _, f := range allObservers {
		st, err := earth.NewSunTimes(f.obs, canonical)
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}
		noon, ok := st.SolarNoon()
		if !ok {
			t.Errorf("%s: SolarNoon not ok", f.name)
			continue
		}
		if noon.Before(st.DayStart()) || noon.After(st.DayEnd()) {
			t.Errorf("%s: noon %v outside day", f.name, noon)
		}
		peak := sun.Position(f.obs, noon).Altitude
		for _, off := range []time.Duration{-30 * time.Minute, -5 * time.Minute, 5 * time.Minute, 30 * time.Minute} {
			when := noon.Add(off)
			if when.Before(st.DayStart()) || when.After(st.DayEnd()) {
				continue
			}
			if a := sun.Position(f.obs, when).Altitude; a > peak+1e-6 {
				t.Errorf("%s: altitude %.5f at %v exceeds noon peak %.5f", f.name, a, when, peak)
			}
		}
	}
}

func TestPolarStates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		obs  astronomy.Observer
		date time.Time
		want earth.PolarState
	}{
		{"svalbard-may", svalbard, canonical, earth.MidnightSun},
		{"svalbard-dec", svalbard, time.Date(1982, 12, 21, 12, 0, 0, 0, time.UTC), earth.PolarNight},
		{"brooklyn-may", brooklyn, canonical, earth.NotPolar},
		{"quito-may", quito, canonical, earth.NotPolar},
	}
	for _, c := range cases {
		st, err := earth.NewSunTimes(c.obs, c.date)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		state, isPolar := st.Polar()
		if state != c.want {
			t.Errorf("%s: polar state = %v, want %v", c.name, state, c.want)
		}
		if isPolar != (c.want != earth.NotPolar) {
			t.Errorf("%s: isPolar = %v, want %v", c.name, isPolar, c.want != earth.NotPolar)
		}
		// Polar days never present a sunrise or sunset band.
		_, hasRise := segmentByLabel(st.Segments(), earth.LabelSunrise)
		if c.want != earth.NotPolar && hasRise {
			t.Errorf("%s: polar day should have no sunrise band", c.name)
		}
	}
}

func TestDayLengthCategories(t *testing.T) {
	t.Parallel()
	// Daylight span from sunrise to sunset for locations that have both.
	daylight := func(obs astronomy.Observer) (time.Duration, bool) {
		st, err := earth.NewSunTimes(obs, canonical)
		if err != nil {
			t.Fatal(err)
		}
		rise, okR := segmentByLabel(st.Segments(), earth.LabelSunrise)
		set, okS := segmentByLabel(st.Segments(), earth.LabelSunset)
		if !okR || !okS {
			return 0, false
		}
		return clockDuration(rise.From, set.To), true
	}
	// Equatorial locations sit near a twelve-hour day year round.
	for _, f := range []struct {
		name string
		obs  astronomy.Observer
	}{{"quito", quito}, {"singapore", singapore}} {
		d, ok := daylight(f.obs)
		if !ok {
			t.Fatalf("%s: expected sunrise and sunset", f.name)
		}
		if d < 11*time.Hour+40*time.Minute || d > 12*time.Hour+30*time.Minute {
			t.Errorf("%s: daylight %v, want ~12h", f.name, d)
		}
	}
	// Ushuaia, deep in the southern autumn, has a short day.
	if d, ok := daylight(ushuaia); !ok || d > 11*time.Hour {
		t.Errorf("ushuaia: daylight %v (ok=%v), want a short day", d, ok)
	}
	// Tromso, seven weeks before the solstice, has a long day.
	if d, ok := daylight(tromso); !ok || d < 18*time.Hour {
		t.Errorf("tromso: daylight %v (ok=%v), want a long day", d, ok)
	}
}

func TestDSTDayLength(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		date time.Time
		want time.Duration
	}{
		{"spring-forward", time.Date(1982, 4, 25, 12, 0, 0, 0, time.UTC), 23 * time.Hour},
		{"fall-back", time.Date(1982, 10, 31, 12, 0, 0, 0, time.UTC), 25 * time.Hour},
	}
	for _, c := range cases {
		st, err := earth.NewSunTimes(brooklyn, c.date)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if st.DayLength() != c.want {
			t.Errorf("%s: day length %v, want %v", c.name, st.DayLength(), c.want)
		}
		var sum float64
		for _, s := range st.Segments() {
			sum += s.Seconds
		}
		if math.Abs(sum-c.want.Seconds()) > 1.0 {
			t.Errorf("%s: segments sum %.1fs, want %.1fs", c.name, sum, c.want.Seconds())
		}
	}
}

// TestMidnightContinuity is the seamless-rollover guarantee: a live clock at
// 23:59:59 and at 00:00:01 must resolve to the same stitched deep-night band, so
// its span and progress fraction are continuous across midnight.
func TestMidnightContinuity(t *testing.T) {
	t.Parallel()
	loc := brooklyn.Location()
	midnight := time.Date(1982, 5, 4, 0, 0, 0, 0, loc)
	before := midnight.Add(-time.Second)
	after := midnight.Add(time.Second)

	segBefore, fracBefore, err := earth.SegmentAt(brooklyn, before)
	if err != nil {
		t.Fatal(err)
	}
	segAfter, fracAfter, err := earth.SegmentAt(brooklyn, after)
	if err != nil {
		t.Fatal(err)
	}
	if segBefore.Label != segAfter.Label {
		t.Errorf("label discontinuity: %q then %q", segBefore.Label, segAfter.Label)
	}
	if !segBefore.From.Equal(segAfter.From) || !segBefore.To.Equal(segAfter.To) {
		t.Errorf("span discontinuity: [%v,%v] then [%v,%v]",
			segBefore.From, segBefore.To, segAfter.From, segAfter.To)
	}
	if !segBefore.From.Before(midnight) || !segBefore.To.After(midnight) {
		t.Errorf("stitched band %v..%v does not straddle midnight %v", segBefore.From, segBefore.To, midnight)
	}
	if fracAfter <= fracBefore {
		t.Errorf("fraction not increasing across midnight: %.8f then %.8f", fracBefore, fracAfter)
	}
	if d := fracAfter - fracBefore; d > 1e-4 {
		t.Errorf("fraction jump across midnight too large: %.8f", d)
	}
}

func TestSegmentAtWithinBand(t *testing.T) {
	t.Parallel()
	loc := brooklyn.Location()
	for _, hm := range []struct{ h, m int }{{2, 0}, {5, 53}, {9, 0}, {12, 52}, {19, 52}, {22, 0}} {
		when := time.Date(1982, 5, 3, hm.h, hm.m, 0, 0, loc)
		seg, frac, err := earth.SegmentAt(brooklyn, when)
		if err != nil {
			t.Fatalf("%02d:%02d: %v", hm.h, hm.m, err)
		}
		if when.Before(seg.From) || !when.Before(seg.To) {
			t.Errorf("%02d:%02d: instant %v outside band [%v,%v]", hm.h, hm.m, when, seg.From, seg.To)
		}
		if frac < 0 || frac >= 1 {
			t.Errorf("%02d:%02d: fraction %.6f out of [0,1)", hm.h, hm.m, frac)
		}
	}
}

// TestSegmentAtMatchesSunTimes checks that for an interior instant SegmentAt
// reports the same band that NewSunTimes resolves for that day.
func TestSegmentAtMatchesSunTimes(t *testing.T) {
	t.Parallel()
	loc := brooklyn.Location()
	when := time.Date(1982, 5, 3, 9, 0, 0, 0, loc) // mid-morning, interior band
	seg, _, err := earth.SegmentAt(brooklyn, when)
	if err != nil {
		t.Fatal(err)
	}
	st, err := earth.NewSunTimes(brooklyn, when)
	if err != nil {
		t.Fatal(err)
	}
	var match earth.Segment
	for _, s := range st.Segments() {
		if s.Contains(when) {
			match = s
			break
		}
	}
	if seg.Label != match.Label || !seg.From.Equal(match.From) || !seg.To.Equal(match.To) {
		t.Errorf("SegmentAt %q[%v,%v] != SunTimes %q[%v,%v]",
			seg.Label, seg.From, seg.To, match.Label, match.From, match.To)
	}
}

func TestConstructorErrors(t *testing.T) {
	t.Parallel()
	bad := astronomy.Observer{Lat: 120, Lng: 0, TZ: time.UTC}
	if _, err := earth.NewSunTimes(bad, canonical); !errors.Is(err, earth.ErrInvalidLatitude) {
		t.Errorf("NewSunTimes bad latitude: err = %v, want ErrInvalidLatitude", err)
	}
	if _, _, err := earth.SegmentAt(bad, canonical); !errors.Is(err, earth.ErrInvalidLatitude) {
		t.Errorf("SegmentAt bad latitude: err = %v, want ErrInvalidLatitude", err)
	}
	empty := earth.Segmentation{Night: "night", Horizon: -0.833}
	if _, err := earth.NewSunTimesWith(brooklyn, canonical, empty); !errors.Is(err, earth.ErrInvalidSegmentation) {
		t.Errorf("empty segmentation: err = %v, want ErrInvalidSegmentation", err)
	}
	descending := earth.Segmentation{
		Night: "night", Horizon: -0.833,
		Levels: []earth.Level{{Altitude: 6}, {Altitude: -6}},
	}
	if _, err := earth.NewSunTimesWith(brooklyn, canonical, descending); !errors.Is(err, earth.ErrInvalidSegmentation) {
		t.Errorf("descending segmentation: err = %v, want ErrInvalidSegmentation", err)
	}
	if _, _, err := earth.SegmentAtWith(brooklyn, canonical, empty); !errors.Is(err, earth.ErrInvalidSegmentation) {
		t.Errorf("SegmentAtWith empty segmentation: err = %v, want ErrInvalidSegmentation", err)
	}
}

// TestCustomSegmentation exercises a consumer-supplied vocabulary: two levels
// dividing the day into deep, twilight, and bright, with the bright band split
// at solar noon.
func TestCustomSegmentation(t *testing.T) {
	t.Parallel()
	seg := earth.Segmentation{
		Night:   "deep",
		Horizon: 0,
		Levels: []earth.Level{
			{Altitude: -6, Rising: "twilight_rising", Setting: "twilight_setting"},
			{Altitude: 10, Rising: "bright_morning", Setting: "bright_afternoon"},
		},
	}
	st, err := earth.NewSunTimesWith(brooklyn, canonical, seg)
	if err != nil {
		t.Fatal(err)
	}
	segs := st.Segments()
	labels := map[string]bool{}
	for _, s := range segs {
		labels[s.Label] = true
	}
	for _, want := range []string{"deep", "twilight_rising", "twilight_setting", "bright_morning", "bright_afternoon"} {
		if !labels[want] {
			t.Errorf("custom segmentation missing label %q; got %v", want, labels)
		}
	}
	// The bright band straddles solar noon, split into morning and afternoon.
	noon, _ := st.SolarNoon()
	morning, _ := segmentByLabel(segs, "bright_morning")
	afternoon, _ := segmentByLabel(segs, "bright_afternoon")
	if !morning.To.Equal(noon) || !afternoon.From.Equal(noon) {
		t.Errorf("bright band not split at noon %v: morning ends %v, afternoon starts %v",
			noon, morning.To, afternoon.From)
	}
}

// TestSegmentAtPolarAndEdges exercises SegmentAt on polar days and at the exact
// day boundary, confirming it resolves without panic and returns a valid band
// and fraction throughout.
func TestSegmentAtPolarAndEdges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		obs  astronomy.Observer
		when time.Time
	}{
		{"svalbard-midnight-sun-noon", svalbard, time.Date(1982, 5, 3, 12, 0, 0, 0, svalbard.Location())},
		{"svalbard-midnight-sun-midnight", svalbard, time.Date(1982, 5, 3, 0, 0, 0, 0, svalbard.Location())},
		{"svalbard-midnight-sun-late", svalbard, time.Date(1982, 5, 3, 23, 59, 59, 0, svalbard.Location())},
		{"svalbard-polar-night", svalbard, time.Date(1982, 12, 21, 3, 0, 0, 0, svalbard.Location())},
		{"brooklyn-exact-midnight", brooklyn, time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())},
	}
	for _, c := range cases {
		seg, frac, err := earth.SegmentAt(c.obs, c.when)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if seg.Label == "" {
			t.Errorf("%s: empty label", c.name)
		}
		if !seg.To.After(seg.From) {
			t.Errorf("%s: non-forward span %v..%v", c.name, seg.From, seg.To)
		}
		if frac < 0 || frac >= 1 {
			t.Errorf("%s: fraction %.6f out of [0,1)", c.name, frac)
		}
	}
}

// TestSegmentAtWithCustom drives the custom-segmentation resolver over a full
// day, confirming every instant lands in a labeled band.
func TestSegmentAtWithCustom(t *testing.T) {
	t.Parallel()
	seg := earth.Segmentation{
		Night:   "dark",
		Horizon: 0,
		Levels: []earth.Level{
			{Altitude: 0, Rising: "up_am", Setting: "up_pm"},
		},
	}
	loc := brooklyn.Location()
	for h := 0; h < 24; h += 3 {
		when := time.Date(1982, 5, 3, h, 0, 0, 0, loc)
		s, frac, err := earth.SegmentAtWith(brooklyn, when, seg)
		if err != nil {
			t.Fatalf("%02d:00: %v", h, err)
		}
		if s.Label == "" || frac < 0 || frac >= 1 {
			t.Errorf("%02d:00: label=%q frac=%.4f", h, s.Label, frac)
		}
	}
}

func TestSegmentContains(t *testing.T) {
	t.Parallel()
	base := time.Date(1982, 5, 3, 10, 0, 0, 0, time.UTC)
	s := earth.Segment{From: base, To: base.Add(time.Hour)}
	if !s.Contains(base) {
		t.Error("From should be inclusive")
	}
	if s.Contains(base.Add(time.Hour)) {
		t.Error("To should be exclusive")
	}
	if s.Contains(base.Add(-time.Second)) {
		t.Error("before From should not be contained")
	}
	if !s.Contains(base.Add(30 * time.Minute)) {
		t.Error("interior instant should be contained")
	}
}

func TestPolarStateString(t *testing.T) {
	t.Parallel()
	cases := map[earth.PolarState]string{
		earth.NotPolar:    "not_polar",
		earth.MidnightSun: "midnight_sun",
		earth.PolarNight:  "polar_night",
	}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("PolarState(%d).String() = %q, want %q", state, got, want)
		}
	}
}

// TestStatelessConcurrency runs concurrent resolutions to guard the promise that
// the package captures no shared mutable state.
func TestStatelessConcurrency(t *testing.T) {
	t.Parallel()
	done := make(chan bool, 8)
	for _, f := range allObservers {
		go func(obs astronomy.Observer) {
			for i := 0; i < 5; i++ {
				if _, err := earth.NewSunTimes(obs, canonical); err != nil {
					t.Error(err)
				}
				if _, _, err := earth.SegmentAt(obs, canonical); err != nil {
					t.Error(err)
				}
			}
			done <- true
		}(f.obs)
	}
	for range allObservers {
		<-done
	}
}
