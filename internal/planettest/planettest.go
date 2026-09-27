// Package planettest holds the checks every planet package runs against the
// committed JPL Horizons fixtures in planet/testdata. Only test files import
// it, so it is never linked into a program.
package planettest

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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/planet"
)

// Greenwich is the observer of the Horizons fixtures.
var Greenwich = astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}

// Module is the module path, for the dependency checks.
const Module = "github.com/Bugs5382/go-astronomy"

// Planets lists the per-planet package names.
var Planets = []string{"mercury", "venus", "mars", "jupiter", "saturn", "uranus", "neptune"}

// root returns the repository root, from this file's own location.
func root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// Row is one row of planet/testdata/horizons-planets.txt.
type Row struct {
	When                   time.Time
	RA, Dec, Az, El        float64
	Mag, Illu, Diam, Delta float64
	Elongation, PhaseAngle float64
}

func rows(t *testing.T, name string) [][]string {
	t.Helper()
	f, err := os.Open(filepath.Join(root(), "planet", "testdata", name)) // #nosec G304 -- a fixture name from this package.
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

// Horizons returns the fixture rows for the named planet.
func Horizons(t *testing.T, planetName string) []Row {
	t.Helper()
	var out []Row
	for _, p := range rows(t, "horizons-planets.txt") {
		if len(p) != 12 {
			t.Fatalf("row %v has %d fields, want 12", p, len(p))
		}
		if p[0] != planetName {
			continue
		}
		when, err := time.Parse(time.RFC3339, p[1])
		if err != nil {
			t.Fatal(err)
		}
		var v [10]float64
		for i := range v {
			if v[i], err = strconv.ParseFloat(p[i+2], 64); err != nil {
				t.Fatal(err)
			}
		}
		out = append(out, Row{when, v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9]})
	}
	if len(out) < 13 {
		t.Fatalf("%s: only %d fixture rows", planetName, len(out))
	}
	return out
}

func sepArcsec(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(lat1*r)*math.Sin(lat2*r) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Acos(math.Min(1, c)) / r * 3600
}

// CheckPosition compares Position with every fixture row for the planet: the
// apparent place within placeTolArcsec, altitude and azimuth within 5 arc
// seconds (they also carry UT1 taken as UTC), the diameter within 0.01 arc
// seconds, the distance within 2e-4 au, the illuminated percentage within
// 0.05, and the elongation and phase angle within 0.02 degrees.
func CheckPosition(t *testing.T, b planet.Body, placeTolArcsec float64) {
	t.Helper()
	var wPlace, wHz float64
	for _, h := range Horizons(t, b.Name()) {
		r, err := b.Position(Greenwich, h.When)
		if err != nil {
			t.Fatal(err)
		}
		place := sepArcsec(r.RA, r.Dec, h.RA, h.Dec)
		hz := sepArcsec(-r.Azimuth, r.Altitude, -h.Az, h.El)
		wPlace, wHz = math.Max(wPlace, place), math.Max(wHz, hz)
		at := h.When.Format(time.RFC3339)
		switch {
		case place > placeTolArcsec:
			t.Errorf("%s: apparent place off Horizons by %.2f arcsec", at, place)
		case hz > 5:
			t.Errorf("%s: altitude and azimuth off Horizons by %.2f arcsec", at, hz)
		case math.Abs(float64(r.Diameter)*3600-h.Diam) > 0.01:
			t.Errorf("%s: diameter %.4f arcsec, Horizons %.4f", at, float64(r.Diameter)*3600, h.Diam)
		case math.Abs(r.DistanceAU-h.Delta) > 2e-4:
			t.Errorf("%s: distance %.6f au, Horizons %.6f", at, r.DistanceAU, h.Delta)
		case math.Abs(r.Illuminated*100-h.Illu) > 0.05:
			t.Errorf("%s: %.3f%% lit, Horizons %.3f%%", at, r.Illuminated*100, h.Illu)
		case math.Abs(r.Elongation-h.Elongation) > 0.02 || math.Abs(r.PhaseAngle-h.PhaseAngle) > 0.02:
			t.Errorf("%s: elongation %.4f and phase %.4f, Horizons %.4f and %.4f", at, r.Elongation, r.PhaseAngle, h.Elongation, h.PhaseAngle)
		}
	}
	t.Logf("%s: worst apparent place %.2f arcsec, altitude and azimuth %.2f arcsec", b.Name(), wPlace, wHz)
}

// CheckMagnitude compares the magnitude with every fixture row within 0.1.
func CheckMagnitude(t *testing.T, b planet.Body) {
	t.Helper()
	worst := 0.0
	for _, h := range Horizons(t, b.Name()) {
		r, err := b.Position(Greenwich, h.When)
		if err != nil {
			t.Fatal(err)
		}
		d := math.Abs(r.Magnitude - h.Mag)
		worst = math.Max(worst, d)
		if d > 0.1 {
			t.Errorf("%s: magnitude %.3f, Horizons %.3f", h.When.Format(time.RFC3339), r.Magnitude, h.Mag)
		}
	}
	t.Logf("%s: worst magnitude %.3f", b.Name(), worst)
}

// CheckRiseSetHorizons compares rise, transit, and set with the Horizons
// fixture rows for the planet, where there are any. Horizons stamps each event
// with the one-minute step at or after it, so the library's instant falls up
// to a minute before the stamp.
func CheckRiseSetHorizons(t *testing.T, b planet.Body) {
	t.Helper()
	checked := 0
	for _, p := range rows(t, "horizons-planets-rts.txt") {
		if p[0] != b.Name() {
			continue
		}
		want, err := time.Parse(time.RFC3339, p[2])
		if err != nil {
			t.Fatal(err)
		}
		from := want.Add(-3 * time.Hour)
		var got time.Time
		var ok bool
		switch p[1] {
		case "r":
			got, ok, err = b.NextRise(Greenwich, from)
		case "s":
			got, ok, err = b.NextSet(Greenwich, from)
		case "t":
			got, ok, err = b.NextTransit(Greenwich, from)
		}
		if err != nil || !ok {
			t.Fatalf("%s %s: %v %v", b.Name(), p[1], ok, err)
		}
		if d := got.Sub(want); d < -65*time.Second || d > 5*time.Second {
			t.Errorf("%s %s: %s, Horizons %s (off %v)", b.Name(), p[1], got.Format(time.RFC3339), p[2], d.Round(time.Second))
		}
		checked++
	}
	t.Logf("%s: %d rise, transit, and set events against Horizons", b.Name(), checked)
}

// CheckRiseSetConsistency checks, for a year of starts at Greenwich, that a
// rise and the following set bracket a transit, that the altitude at rise and
// set is the horizon altitude, and that the hour angle at transit is zero.
func CheckRiseSetConsistency(t *testing.T, b planet.Body) {
	t.Helper()
	for d := 0; d < 365; d += 37 {
		from := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
		rise, ok, err := b.NextRise(Greenwich, from)
		if err != nil || !ok {
			t.Fatalf("NextRise from %s: %v %v", from, ok, err)
		}
		set, ok, err := b.NextSet(Greenwich, rise)
		if err != nil || !ok {
			t.Fatalf("NextSet from %s: %v %v", rise, ok, err)
		}
		transit, ok, err := b.NextTransit(Greenwich, rise)
		if err != nil || !ok {
			t.Fatalf("NextTransit from %s: %v %v", rise, ok, err)
		}
		if !rise.After(from) || !transit.Before(set) {
			t.Errorf("from %s: rise %s, transit %s, set %s out of order", from, rise, transit, set)
		}
		for _, e := range []time.Time{rise, set} {
			r, _ := b.Position(Greenwich, e)
			if math.Abs(r.Altitude-planet.HorizonAltitude) > 0.01 {
				t.Errorf("altitude %.4f at %s, want %.4f", r.Altitude, e, planet.HorizonAltitude)
			}
		}
		up, _ := b.Position(Greenwich, transit)
		before, _ := b.Position(Greenwich, transit.Add(-2*time.Minute))
		after, _ := b.Position(Greenwich, transit.Add(2*time.Minute))
		if up.Altitude < before.Altitude || up.Altitude < after.Altitude {
			t.Errorf("transit at %s is not the highest point", transit)
		}
	}
}

// CheckHeliocentric checks the planet's heliocentric vector against Earth's:
// their difference has the length of the geometric Earth-planet distance,
// which differs from Position's by the planet's motion during the light-time.
func CheckHeliocentric(t *testing.T, b planet.Body) {
	t.Helper()
	when := time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)
	e, p := planet.EarthHeliocentric(when).Vector(), b.Heliocentric(when).Vector()
	dx, dy, dz := p[0]-e[0], p[1]-e[1], p[2]-e[2]
	d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	r, err := b.Position(Greenwich, when)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(d-r.DistanceAU) > 2e-3 {
		t.Errorf("vector difference %.5f au, Position distance %.5f", d, r.DistanceAU)
	}
	h := b.Heliocentric(when)
	if !(h.Lon >= 0 && h.Lon < 360) || h.DistanceAU <= 0 {
		t.Errorf("heliocentric %+v", h)
	}
}

// CheckObserverErrors checks the coded errors for an out-of-range observer.
func CheckObserverErrors(t *testing.T, b planet.Body) {
	t.Helper()
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := b.Position(astronomy.Observer{Lat: 91}, when); !errors.Is(err, planet.ErrInvalidLatitude) {
		t.Errorf("latitude 91: %v", err)
	}
	if _, err := b.Position(astronomy.Observer{Lng: 181}, when); !errors.Is(err, planet.ErrInvalidLongitude) {
		t.Errorf("longitude 181: %v", err)
	}
	if _, _, err := b.NextRise(astronomy.Observer{Lat: 91}, when); !errors.Is(err, planet.ErrInvalidLatitude) {
		t.Errorf("NextRise latitude 91: %v", err)
	}
}

// CheckNearSun checks the flag follows the elongation and the planet's limit.
func CheckNearSun(t *testing.T, b planet.Body) {
	t.Helper()
	for h := 0; h < 400*24; h += 97 {
		when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(h) * time.Hour)
		r, err := b.Position(Greenwich, when)
		if err != nil {
			t.Fatal(err)
		}
		if r.NearSun != (r.Elongation < b.NearSunElongation()) {
			t.Fatalf("%s: NearSun %v with elongation %.2f", when, r.NearSun, r.Elongation)
		}
	}
}

// CheckIsolated checks that the package links no other planet's package, and
// so none of their tables, and not planet/all.
func CheckIsolated(t *testing.T, name string) {
	t.Helper()
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("no go command: %v", err)
	}
	cmd := exec.Command(gobin, "list", "-deps", "-f", "{{.ImportPath}}", Module+"/planet/"+name) // #nosec G204 -- fixed arguments.
	cmd.Dir = root()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	deps := strings.Fields(string(out))
	if len(deps) < 5 {
		t.Fatalf("go list gave %v", deps)
	}
	for _, d := range deps {
		if d == Module+"/planet/all" {
			t.Errorf("planet/%s depends on planet/all", name)
		}
		for _, other := range Planets {
			if other != name && d == Module+"/planet/"+other {
				t.Errorf("planet/%s depends on planet/%s", name, other)
			}
		}
	}
}

// CheckHeight checks the observer's height: an omitted height gives exactly
// the answers of an explicit zero, NaN and infinity are rejected with
// astronomy.ErrInvalidHeight, and a height lowers the horizon by the dip and
// thins the refraction by the standard atmosphere, so the planet rises earlier
// and sets later, by the dip's worth of motion less the refraction lost.
func CheckHeight(t *testing.T, b planet.Body) {
	t.Helper()
	when := time.Date(2027, 5, 1, 0, 0, 0, 0, time.UTC)
	omitted := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	zero := omitted
	zero.Height = astronomy.Meters(0)
	a, errA := b.Position(omitted, when)
	z, errZ := b.Position(zero, when)
	if errA != nil || errZ != nil || a != z {
		t.Errorf("omitted height %+v, explicit zero %+v", a, z)
	}
	for _, h := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		bad := omitted
		bad.Height = astronomy.Meters(h)
		if _, err := b.Position(bad, when); !errors.Is(err, astronomy.ErrInvalidHeight) {
			t.Errorf("Position with height %v: %v", h, err)
		}
		if _, _, err := b.NextRise(bad, when); !errors.Is(err, astronomy.ErrInvalidHeight) {
			t.Errorf("NextRise with height %v: %v", h, err)
		}
	}
	high := omitted
	high.Height = astronomy.Feet(5000)
	r0, _, _ := b.NextRise(omitted, when)
	r1, _, _ := b.NextRise(high, r0.Add(-time.Hour))
	s0, _, _ := b.NextSet(omitted, when)
	s1, _, _ := b.NextSet(high, s0.Add(-time.Hour))
	// The dip at 1524 m is 1.145 degrees, less 0.08 of refraction the thinner
	// air does not give; at Denver's latitude a planet climbs
	// through the horizon at 7 to 13 degrees an hour, so 5 to 10 minutes.
	for name, d := range map[string]time.Duration{"rise earlier": r0.Sub(r1), "set later": s1.Sub(s0)} {
		if d < 4*time.Minute || d > 12*time.Minute {
			t.Errorf("5000 ft moved the %s by %v, want 4 to 12 minutes", name, d)
		}
	}
	// At the rise from height the centre sits on the scaled horizon: the
	// standard refraction thinned by the air at 1524 m, lowered by the dip.
	if r, err := b.Position(high, r1); err == nil {
		want := planet.HorizonAltitude*earth.StandardAtmosphere.Factor(high.Height) - earth.HorizonDip(high.Height)
		if math.Abs(r.Altitude-want) > 0.01 {
			t.Errorf("altitude at the rise from 5000 ft %.4f, want %.4f", r.Altitude, want)
		}
	}
	p0, _ := b.Position(omitted, when)
	p1, _ := b.Position(high, when)
	if d := sepArcsec(p0.RA, p0.Dec, p1.RA, p1.Dec); d > 0.1 {
		t.Errorf("the height moved the place by %.3f arcsec, want well under 0.1", d)
	}
}
