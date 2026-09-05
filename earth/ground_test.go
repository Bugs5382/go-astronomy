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
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// groundObs is New York on a spring day: the Sun climbs well above the horizon
// at noon and sinks well below the astronomical threshold at night, so the
// darkness factor exercises both clamped endpoints and the full ramp between.
var groundObs = astronomy.Observer{Lat: 40.7128, Lng: -74.0060}

func TestNightDarknessZeroAtSolarNoon(t *testing.T) {
	st, err := earth.NewSunTimes(groundObs, time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewSunTimes: %v", err)
	}
	noon, _ := st.SolarNoon()
	if d := earth.NightDarkness(groundObs, noon); d != 0 {
		t.Errorf("darkness at solar noon = %v, want 0", d)
	}
}

func TestNightDarknessOneDeepNight(t *testing.T) {
	// Local solar midnight: the Sun is far below -18 degrees, full darkness.
	midnight := time.Date(2026, 3, 26, 5, 0, 0, 0, time.UTC) // ~00:00 local (UTC-5)
	if d := earth.NightDarkness(groundObs, midnight); d != 1 {
		t.Errorf("darkness at deep night = %v, want 1", d)
	}
}

func TestNightDarknessRangeAndMonotonic(t *testing.T) {
	// Sweep from solar noon to the following solar midnight. The Sun's altitude
	// falls monotonically across this window, so the darkness factor must rise
	// monotonically, staying within [0, 1].
	st, err := earth.NewSunTimes(groundObs, time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewSunTimes: %v", err)
	}
	noon, _ := st.SolarNoon()

	prev := -1.0
	sawRampInterior := false
	for step := 0; step <= 72; step++ { // 12 hours in 10-minute steps
		when := noon.Add(time.Duration(step) * 10 * time.Minute)
		d := earth.NightDarkness(groundObs, when)
		if d < 0 || d > 1 {
			t.Fatalf("%v: darkness %v out of [0,1]", when, d)
		}
		if d < prev-1e-12 {
			t.Errorf("%v: darkness %v decreased from %v", when, d, prev)
		}
		if d > 0 && d < 1 {
			sawRampInterior = true
		}
		prev = d
	}
	if !sawRampInterior {
		t.Error("expected at least one twilight sample strictly between 0 and 1")
	}
	if prev != 1 {
		t.Errorf("darkness at end of sweep = %v, want 1", prev)
	}
}

func TestNightDarknessSmoothClampedEndpoints(t *testing.T) {
	// Above the day threshold the factor is exactly 0; below the night
	// threshold it is exactly 1. Confirm the clamps hold across a full day.
	for step := 0; step < 288; step++ { // 24 hours in 5-minute steps
		when := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC).Add(time.Duration(step) * 5 * time.Minute)
		alt := earth.SunPosition(groundObs, when).Altitude
		d := earth.NightDarkness(groundObs, when)
		switch {
		case alt >= 0 && d != 0:
			t.Errorf("%v: alt %v >= 0 but darkness %v != 0", when, alt, d)
		case alt <= -18 && d != 1:
			t.Errorf("%v: alt %v <= -18 but darkness %v != 1", when, alt, d)
		}
	}
}
