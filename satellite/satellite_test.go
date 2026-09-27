package satellite_test

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
	"github.com/Bugs5382/go-astronomy/satellite"
)

// The synthetic ISS element set the Python and Skyfield fixtures were built
// from (see testdata/iss_crosscheck.py).
const (
	iss1 = "1 25544U 98067A   26269.51782528  .00016717  00000-0  30306-3 0  9990"
	iss2 = "2 25544  51.6416 247.4627 0006703 130.5360 325.0288 15.49450470 12348"
)

func iss(t *testing.T) satellite.Elements {
	t.Helper()
	e, err := satellite.ParseTLE(iss1, iss2)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func fields(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out [][]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.Fields(line))
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func num(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestParseOMMAgainstPython reads the JSON and XML OMM samples and checks the
// propagated states against python-sgp4's own OMM reader.
func TestParseOMMAgainstPython(t *testing.T) {
	t.Parallel()
	sets := map[string]satellite.Elements{}
	for _, name := range []string{"omm-vanguard.json", "omm-vanguard.xml"} {
		f, err := os.Open("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		els, err := satellite.ParseOMM(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(els) != 1 || els[0].SatNum != 5 || els[0].IntlDesignator != "58002B" {
			t.Fatalf("%s: parsed %+v", name, els)
		}
		sets[name] = els[0]
	}
	rows := fields(t, "testdata/omm_python.txt")
	for _, r := range rows {
		e := sets[r[0]]
		pos, vel, err := e.PropagateMinutes(num(t, r[1]))
		if err != nil {
			t.Fatal(err)
		}
		for i := range 3 {
			if d := math.Abs(pos[i] - num(t, r[2+i])); d > 1e-6 {
				t.Errorf("%s %s min: position[%d] off by %.2e km", r[0], r[1], i, d)
			}
			if d := math.Abs(vel[i] - num(t, r[5+i])); d > 1e-9 {
				t.Errorf("%s %s min: velocity[%d] off by %.2e km/s", r[0], r[1], i, d)
			}
		}
	}
	if len(rows) != 8 {
		t.Fatalf("%d reference rows, want 8", len(rows))
	}
	want := time.Date(2025, 2, 14, 14, 36, 48, 662784000, time.UTC)
	if got := sets["omm-vanguard.json"].Epoch(); !got.Equal(want) {
		t.Errorf("epoch %s, want %s", got, want)
	}
}

// TestParseOMMErrors checks malformed and unsuitable messages are rejected
// with the coded ErrMalformedOMM.
func TestParseOMMErrors(t *testing.T) {
	t.Parallel()
	base := `{"OBJECT_ID":"1958-002B","EPOCH":"2025-02-14T14:36:48.662784","MEAN_MOTION":10.85873516,` +
		`"ECCENTRICITY":0.1841322,"INCLINATION":34.2493,"RA_OF_ASC_NODE":19.2327,"ARG_OF_PERICENTER":100.1057,` +
		`"MEAN_ANOMALY":281.1229,"NORAD_CAT_ID":5,"BSTAR":0.00035436`
	cases := map[string]string{
		"empty":        "  ",
		"not json/xml": "1 25544U",
		"bad json":     "[{",
		"no records":   "[]",
		"missing":      `{"EPOCH":"2025-02-14T14:36:48"}`,
		"bad epoch":    strings.Replace(base, "2025-02-14T14:36:48.662784", "yesterday", 1) + "}",
		"not sgp4":     base + `,"MEAN_ELEMENT_THEORY":"DSST"}`,
		"not teme":     base + `,"REF_FRAME":"GCRF"}`,
		"bad number":   strings.Replace(base, "0.1841322", `"x"`, 1) + "}",
	}
	for name, msg := range cases {
		_, err := satellite.ParseOMM(strings.NewReader(msg))
		code, _ := apperr.Code(err)
		if !errors.Is(err, satellite.ErrMalformedOMM) || code != astronomy.CodeInvalidElements {
			t.Errorf("%s: error = %v (code %d)", name, err, code)
		}
	}
	if _, err := satellite.ParseOMM(strings.NewReader(base + "}")); err != nil {
		t.Errorf("single object: %v", err)
	}
}

// TestPassesAgainstSkyfield checks rise, peak, set, peak altitude, and shadow
// entry and exit against Skyfield for the same element set, two observers,
// and two days (testdata/passes-skyfield.txt).
func TestPassesAgainstSkyfield(t *testing.T) {
	t.Parallel()
	e := iss(t)
	from := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	to := from.Add(48 * time.Hour)
	got := map[string][]satellite.Pass{}
	rows := fields(t, "testdata/passes-skyfield.txt")
	parse := func(s string) time.Time {
		if s == "-" {
			return time.Time{}
		}
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// Observed worst: 0.19 s on rise, peak, and set, 0.05 s on shadow
	// crossings, and 0.007 degrees of peak altitude.
	const tol = time.Second
	var worstT, worstShadow time.Duration
	var worstAlt float64
	defer func() {
		t.Logf("worst rise/peak/set %v, shadow %v, peak altitude %.4f degrees", worstT, worstShadow, worstAlt)
	}()
	near := func(a, b time.Time) bool {
		if a.IsZero() || b.IsZero() {
			return a.IsZero() && b.IsZero()
		}
		return a.Sub(b).Abs() <= tol
	}
	for _, r := range rows {
		site := r[0]
		if got[site] == nil {
			obs := astronomy.Observer{Lat: num(t, r[1]), Lng: num(t, r[2]), TZ: time.UTC}
			p, err := satellite.Passes(obs, e, from, to, satellite.DefaultPassOptions())
			if err != nil {
				t.Fatal(err)
			}
			got[site] = p
		}
		rise := parse(r[3])
		var match *satellite.Pass
		for i := range got[site] {
			if got[site][i].Rise.Time.Sub(rise).Abs() < time.Minute {
				match = &got[site][i]
			}
		}
		if match == nil {
			t.Errorf("%s: no pass rising near %s", site, r[3])
			continue
		}
		if !near(match.Rise.Time, rise) || !near(match.Peak.Time, parse(r[4])) || !near(match.Set.Time, parse(r[5])) {
			t.Errorf("%s %s: rise/peak/set %s %s %s, Skyfield %s %s %s", site, r[3],
				match.Rise.Time.Format(time.TimeOnly), match.Peak.Time.Format(time.TimeOnly), match.Set.Time.Format(time.TimeOnly), r[3], r[4], r[5])
		}
		worstT = max(worstT, match.Rise.Time.Sub(rise).Abs(), match.Peak.Time.Sub(parse(r[4])).Abs(), match.Set.Time.Sub(parse(r[5])).Abs())
		if !match.ShadowEntry.IsZero() && r[7] != "-" {
			worstShadow = max(worstShadow, match.ShadowEntry.Sub(parse(r[7])).Abs())
		}
		if !match.ShadowExit.IsZero() && r[8] != "-" {
			worstShadow = max(worstShadow, match.ShadowExit.Sub(parse(r[8])).Abs())
		}
		worstAlt = math.Max(worstAlt, math.Abs(match.Peak.Altitude-num(t, r[6])))
		if d := math.Abs(match.Peak.Altitude - num(t, r[6])); d > 0.02 {
			t.Errorf("%s %s: peak altitude %.4f, Skyfield %s", site, r[3], match.Peak.Altitude, r[6])
		}
		if !near(match.ShadowEntry, parse(r[7])) || !near(match.ShadowExit, parse(r[8])) {
			t.Errorf("%s %s: shadow entry/exit %s %s, Skyfield %s %s", site, r[3], match.ShadowEntry, match.ShadowExit, r[7], r[8])
		}
	}
	for site, p := range got {
		n := 0
		for _, r := range rows {
			if r[0] == site {
				n++
			}
		}
		if len(p) != n {
			t.Errorf("%s: %d passes, Skyfield %d", site, len(p), n)
		}
	}
}

// TestVisiblePass checks the visibility flag on the Denver evening pass of
// 2026-09-26 01:18 UTC (19:18 MDT): the Sun is below -6 degrees at Denver and
// the station is sunlit until it enters the shadow at 01:27:05.
func TestVisiblePass(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	opt := satellite.DefaultPassOptions()
	opt.StdMagnitude = satellite.ISSStandardMagnitude
	from := time.Date(2026, 9, 26, 1, 0, 0, 0, time.UTC)
	passes, err := satellite.Passes(obs, iss(t), from, from.Add(time.Hour), opt)
	if err != nil || len(passes) != 1 {
		t.Fatalf("passes = %v, %v", passes, err)
	}
	p := passes[0]
	if !p.Visible || p.VisibleTo.After(p.ShadowEntry.Add(time.Second)) || p.VisibleFrom.Before(p.Rise.Time) {
		t.Errorf("visible %v from %s to %s, shadow entry %s", p.Visible, p.VisibleFrom, p.VisibleTo, p.ShadowEntry)
	}
	if m := p.Peak.Magnitude; math.IsNaN(m) || m > 0 || m < -4 {
		t.Errorf("peak magnitude %.2f, want a bright pass between -4 and 0", m)
	}
	if !math.IsInf(p.Set.Magnitude, 1) {
		t.Errorf("magnitude after shadow entry %.2f, want +Inf", p.Set.Magnitude)
	}
}

// TestMagnitudeStandard checks the standard magnitude is reproduced at 1000
// km and a 90 degree phase angle, and that a fuller phase is brighter.
func TestMagnitudeStandard(t *testing.T) {
	t.Parallel()
	l := satellite.Look{RangeKm: 1000, PhaseAngle: 90, Sunlit: true}
	if m := l.Magnitude(-1.8); math.Abs(m+1.8) > 1e-9 {
		t.Errorf("magnitude %.4f at the standard geometry, want -1.8", m)
	}
	l.PhaseAngle = 30
	if l.Magnitude(-1.8) >= -1.8 {
		t.Error("a fuller phase should be brighter")
	}
	l.Sunlit = false
	if !math.IsInf(l.Magnitude(-1.8), 1) {
		t.Error("a satellite in shadow should have no magnitude")
	}
}

// TestPositionBasics checks a Look against the pass geometry and the input
// validation.
func TestPositionBasics(t *testing.T) {
	t.Parallel()
	e := iss(t)
	obs := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	peak := time.Date(2026, 9, 26, 4, 5, 3, 457000000, time.UTC) // Skyfield: 88.60 degrees
	l, err := satellite.Position(obs, e, peak)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(l.Altitude-88.5994) > 0.05 || l.RangeKm < 350 || l.RangeKm > 450 {
		t.Errorf("look %+v", l)
	}
	if l.AltitudeKm < 350 || l.AltitudeKm > 450 || math.Abs(l.Latitude-obs.Lat) > 1 {
		t.Errorf("sub-satellite point %.2f %.2f at %.1f km", l.Latitude, l.Longitude, l.AltitudeKm)
	}
	if math.Abs(l.RangeRateKmS) > 0.2 {
		t.Errorf("range rate %.3f km/s at the peak, want about zero", l.RangeRateKmS)
	}
	if got := e.Age(e.Epoch().Add(36 * time.Hour)); got != 36*time.Hour {
		t.Errorf("Age = %v", got)
	}
	if _, err := satellite.Position(astronomy.Observer{Lat: 91}, e, peak); !errors.Is(err, satellite.ErrInvalidLatitude) {
		t.Errorf("latitude 91: %v", err)
	}
	for _, w := range [][2]time.Time{{peak, peak}, {peak, peak.Add(-time.Hour)}, {peak, peak.Add(32 * 24 * time.Hour)}} {
		_, err := satellite.Passes(obs, e, w[0], w[1], satellite.DefaultPassOptions())
		code, _ := apperr.Code(err)
		if !errors.Is(err, satellite.ErrInvalidPassWindow) || code != astronomy.CodeInvalidPassWindow {
			t.Errorf("window %v: %v", w, err)
		}
	}
}
