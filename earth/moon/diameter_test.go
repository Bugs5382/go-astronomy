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
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// horizonsDiameterRow is one row of testdata/horizons-moon-diameter.txt.
type horizonsDiameterRow struct {
	obs       astronomy.Observer
	when      time.Time
	elevation float64 // degrees, airless
	diameter  float64 // degrees
}

// loadHorizonsDiameters reads the committed JPL Horizons fixture. The file
// header records the query and the DE441 source.
func loadHorizonsDiameters(t *testing.T) []horizonsDiameterRow {
	t.Helper()
	f, err := os.Open("testdata/horizons-moon-diameter.txt")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()
	var rows []horizonsDiameterRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if len(p) != 6 {
			t.Fatalf("fixture row %q: want 6 fields", line)
		}
		num := func(s string) float64 {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				t.Fatalf("fixture row %q: %v", line, err)
			}
			return v
		}
		when, err := time.Parse(time.RFC3339, p[2])
		if err != nil {
			t.Fatalf("fixture row %q: %v", line, err)
		}
		rows = append(rows, horizonsDiameterRow{
			obs:       astronomy.Observer{Lat: num(p[0]), Lng: num(p[1]), TZ: time.UTC},
			when:      when,
			elevation: num(p[3]),
			diameter:  num(p[4]) / 3600,
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("fixture has no rows")
	}
	return rows
}

// TestPositionDiameterIsTopocentric checks Position's Diameter against the
// topocentric angular diameter JPL Horizons reports for the same observer and
// instant (issue 44). The tolerance of 0.0002 degrees (0.7 arc seconds) sits
// well inside the up to 0.01 degree error of a diameter taken from the
// geocentric distance at high altitude.
func TestPositionDiameterIsTopocentric(t *testing.T) {
	t.Parallel()
	const tol = 0.0002
	for _, row := range loadHorizonsDiameters(t) {
		pos, err := moon.Position(row.obs, row.when)
		if err != nil {
			t.Fatalf("Position(%v, %s): %v", row.obs, row.when, err)
		}
		if d := float64(pos.Diameter) - row.diameter; d > tol || d < -tol {
			t.Errorf("%s at %.2f,%.2f (alt %.1f): diameter %.5f, Horizons %.5f (off %.5f)",
				row.when.Format(time.RFC3339), row.obs.Lat, row.obs.Lng, row.elevation,
				float64(pos.Diameter), row.diameter, d)
		}
	}
}

// TestPositionDiameterGrowsWithAltitude checks the property behind issue 44: at
// one instant the geocentric distance is fixed, so observers who see the Moon
// higher are nearer to it and must see a larger disc.
func TestPositionDiameterGrowsWithAltitude(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	type sample struct{ alt, diam float64 }
	var samples []sample
	for lat := -80.0; lat <= 80; lat += 10 {
		pos, err := moon.Position(astronomy.Observer{Lat: lat, Lng: 0}, when)
		if err != nil {
			t.Fatalf("Position: %v", err)
		}
		samples = append(samples, sample{pos.Altitude, float64(pos.Diameter)})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].alt < samples[j].alt })
	for i := 1; i < len(samples); i++ {
		if samples[i].diam <= samples[i-1].diam {
			t.Errorf("diameter %.6f at alt %.2f is not larger than %.6f at alt %.2f",
				samples[i].diam, samples[i].alt, samples[i-1].diam, samples[i-1].alt)
		}
	}
}
