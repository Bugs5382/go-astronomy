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
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// TestHorizonDip checks the dip of the sea horizon, 1.76 arc minutes times the
// square root of the height in metres (the Nautical Almanac form, which
// includes standard terrestrial refraction), and its clamp at sea level.
func TestHorizonDip(t *testing.T) {
	t.Parallel()
	cases := []struct{ h, want float64 }{
		{0, 0},
		{-430, 0},
		{1, 1.76 / 60},
		{100, 17.6 / 60},
		{1609, 1.76 * math.Sqrt(1609) / 60},
		{9000, 1.76 * math.Sqrt(9000) / 60},
	}
	for _, c := range cases {
		if got := earth.HorizonDip(astronomy.Meters(c.h)); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("HorizonDip(%v) = %.6f, want %.6f", c.h, got, c.want)
		}
	}
}

// TestInvalidHeight checks that a NaN or infinite height is rejected with the
// coded ErrInvalidHeight by every function that validates an observer, and
// that any real height, from the Dead Sea to cruising altitude, is accepted.
func TestInvalidHeight(t *testing.T) {
	t.Parallel()
	day := time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)
	for _, h := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, Height: astronomy.Meters(h)}
		_, err := earth.NewSunTimes(obs, day)
		if code, _ := apperr.Code(err); !errors.Is(err, astronomy.ErrInvalidHeight) || code != astronomy.CodeInvalidHeight {
			t.Errorf("NewSunTimes(height %v) error = %v, want ErrInvalidHeight with code %d", h, err, astronomy.CodeInvalidHeight)
		}
		if _, _, err := earth.SegmentAt(obs, day); !errors.Is(err, astronomy.ErrInvalidHeight) {
			t.Errorf("SegmentAt(height %v) error = %v, want ErrInvalidHeight", h, err)
		}
		if _, err := earth.SeasonAt(obs, day); !errors.Is(err, astronomy.ErrInvalidHeight) {
			t.Errorf("SeasonAt(height %v) error = %v, want ErrInvalidHeight", h, err)
		}
	}
	for _, h := range []astronomy.Height{astronomy.Meters(-430), astronomy.SeaLevel, astronomy.Meters(8849), astronomy.Feet(36000), astronomy.Meters(-1e6)} {
		obs := astronomy.Observer{Lat: 27.99, Lng: 86.93, Height: h}
		if _, err := earth.NewSunTimes(obs, day); err != nil {
			t.Errorf("NewSunTimes(height %v) = %v, want no error", h, err)
		}
	}
}

// TestOmittedHeightIsSeaLevel checks that leaving Height out gives exactly
// the answers of an explicit zero.
func TestOmittedHeightIsSeaLevel(t *testing.T) {
	t.Parallel()
	omitted := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC}
	explicit := omitted
	explicit.Height = astronomy.Meters(0)
	date := time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)
	a, err := earth.NewSunTimes(omitted, date)
	if err != nil {
		t.Fatal(err)
	}
	b, err := earth.NewSunTimes(explicit, date)
	if err != nil {
		t.Fatal(err)
	}
	sa, sb := a.Segments(), b.Segments()
	if len(sa) != len(sb) {
		t.Fatalf("%d and %d segments", len(sa), len(sb))
	}
	for i := range sa {
		if sa[i] != sb[i] {
			t.Errorf("segment %d: %+v and %+v", i, sa[i], sb[i])
		}
	}
	if earth.SunPosition(omitted, date) != earth.SunPosition(explicit, date) {
		t.Error("SunPosition differs between an omitted and an explicit zero height")
	}
}

// TestFakeHighHeightIsAccepted checks a height that does not match the
// ground is used as given: New York at 5000 ft (1524 m) moves sunrise earlier
// by the dip, as a mountain there would.
func TestFakeHighHeightIsAccepted(t *testing.T) {
	t.Parallel()
	sea := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC}
	high := sea
	high.Height = astronomy.Feet(5000)
	if high.Height.Meters() != 1524 {
		t.Fatalf("5000 ft = %v m", high.Height.Meters())
	}
	date := time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)
	rise := func(obs astronomy.Observer) time.Time {
		day, err := earth.NewSunTimes(obs, date)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range day.Segments() {
			if s.Label == earth.LabelSunrise {
				return s.From
			}
		}
		t.Fatal("no sunrise")
		return time.Time{}
	}
	if d := rise(sea).Sub(rise(high)); d < 5*time.Minute || d > 9*time.Minute {
		t.Errorf("5000 ft moved sunrise earlier by %v, want 5 to 9 minutes", d)
	}
}

// tvhEvent is one row of testdata/horizons-tvh.txt.
type tvhEvent struct {
	site   string
	obs    astronomy.Observer
	rising bool
	when   time.Time
}

func loadTVH(t *testing.T) []tvhEvent {
	t.Helper()
	f, err := os.Open("testdata/horizons-tvh.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	num := func(s string) float64 {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	var out []tvhEvent
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if len(p) != 6 {
			t.Fatalf("row %q has %d fields, want 6", line, len(p))
		}
		when, err := time.Parse(time.RFC3339, p[5])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, tvhEvent{
			site:   p[0],
			obs:    astronomy.Observer{Lat: num(p[1]), Lng: num(p[2]), Height: astronomy.Meters(num(p[3])), TZ: time.UTC},
			rising: p[4] == "r",
			when:   when,
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) < 20 {
		t.Fatalf("only %d fixture rows", len(out))
	}
	return out
}

// libraryRiseSet returns the sunrise or sunset nearest to near: the start of
// the sunrise band or the end of the sunset band on the civil day around it.
func libraryRiseSet(t *testing.T, obs astronomy.Observer, near time.Time, rising bool) time.Time {
	t.Helper()
	var best time.Time
	for _, d := range []int{-1, 0, 1} {
		day, err := earth.NewSunTimes(obs, near.AddDate(0, 0, d))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range day.Segments() {
			var edge time.Time
			switch {
			case rising && s.Label == earth.LabelSunrise:
				edge = s.From
			case !rising && s.Label == earth.LabelSunset:
				edge = s.To
			default:
				continue
			}
			if best.IsZero() || edge.Sub(near).Abs() < best.Sub(near).Abs() {
				best = edge
			}
		}
	}
	return best
}

// TestRiseSetAgainstTrueVisualHorizon checks sunrise and sunset for Denver
// (1609 m) and La Paz (3640 m) against the JPL Horizons true visual horizon, at
// the site's height and at sea level, across the solstices and an equinox
// (issue 52). At height the dip moves each event by 7 to 9 minutes, so a
// height-less answer fails every height row.
//
// Horizons stamps each event with the one-minute step at or after it, so the
// library's instant should fall up to a minute before the stamp: at sea level
// it does, in [-60 s, 0]. At height there is a second, known difference.
// Horizons' true visual horizon uses the geometric dip, 1.93 * sqrt(h) arc
// minutes, while this library uses the observed dip of the Nautical Almanac,
// 1.76 * sqrt(h), which includes terrestrial refraction. The smaller dip rises
// later and sets earlier, by up to about 50 s at these heights and latitudes.
func TestRiseSetAgainstTrueVisualHorizon(t *testing.T) {
	t.Parallel()
	const quantum = time.Minute
	const slack = 5 * time.Second
	for _, e := range loadTVH(t) {
		got := libraryRiseSet(t, e.obs, e.when, e.rising)
		lo, hi := -quantum-slack, slack
		if e.obs.Height.Meters() > 0 {
			// The smaller, observed dip: sunrise later, sunset earlier.
			const dipGap = 70 * time.Second
			if e.rising {
				hi += dipGap
			} else {
				lo -= dipGap
			}
		}
		if d := got.Sub(e.when); d < lo || d > hi {
			t.Errorf("%s %.0f m rising=%v: %s, Horizons %s (off %v, want %v to %v)", e.site, e.obs.Height.Meters(), e.rising,
				got.Format(time.RFC3339), e.when.Format(time.RFC3339), d.Round(time.Second), lo, hi)
		}
	}
}

// TestDipMovesOnlySunriseAndSunsetByDefault checks the USNO convention the
// default segmentation follows: height moves the sunrise and sunset boundaries
// but not the civil, nautical, or astronomical twilight boundaries, which are
// depressions below the astronomical horizon. WithTwilightDip opts every level
// in.
func TestDipMovesOnlySunriseAndSunsetByDefault(t *testing.T) {
	t.Parallel()
	date := time.Date(2027, 3, 20, 12, 0, 0, 0, time.UTC)
	denver, err := time.LoadLocation("America/Denver")
	if err != nil {
		t.Skip("no time zone data:", err)
	}
	sea := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: denver}
	high := sea
	high.Height = astronomy.Meters(1609)

	starts := func(obs astronomy.Observer, seg earth.Segmentation) map[string]time.Time {
		day, err := earth.NewSunTimesWith(obs, date, seg)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]time.Time{}
		for _, s := range day.Segments() {
			if _, seen := out[s.Label]; !seen {
				out[s.Label] = s.From
			}
		}
		return out
	}
	s0 := starts(sea, earth.DefaultSegmentation)
	s1 := starts(high, earth.DefaultSegmentation)
	for _, label := range []string{earth.LabelAstronomicalDawn, earth.LabelNauticalDawn, earth.LabelCivilDawn} {
		if d := s1[label].Sub(s0[label]); d.Abs() > time.Second {
			t.Errorf("%s moved by %v with height, want unmoved", label, d)
		}
	}
	for _, label := range []string{earth.LabelSunrise, earth.LabelGoldenHour} {
		if d := s0[label].Sub(s1[label]); d < 5*time.Minute || d > 10*time.Minute {
			t.Errorf("%s moved earlier by %v with height, want 5 to 10 minutes", label, d)
		}
	}

	all := earth.DefaultSegmentation.WithTwilightDip()
	s2 := starts(high, all)
	for _, label := range []string{earth.LabelAstronomicalDawn, earth.LabelNauticalDawn, earth.LabelCivilDawn, earth.LabelSunrise} {
		if d := s0[label].Sub(s2[label]); d < 5*time.Minute || d > 12*time.Minute {
			t.Errorf("WithTwilightDip: %s moved earlier by %v, want 5 to 12 minutes", label, d)
		}
	}
	for _, l := range earth.DefaultSegmentation.Levels {
		if l.Altitude < -1 && l.DipCorrected {
			t.Errorf("WithTwilightDip changed the shared DefaultSegmentation")
		}
	}
}
