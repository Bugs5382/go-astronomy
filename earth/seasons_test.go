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
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

var (
	northObs = astronomy.Observer{Lat: 40.7128, Lng: -74.0060}  // New York
	southObs = astronomy.Observer{Lat: -33.8688, Lng: 151.2093} // Sydney
)

func seasonAt(t *testing.T, obs astronomy.Observer, when time.Time) earth.SeasonInfo {
	t.Helper()
	info, err := earth.SeasonAt(obs, when)
	if err != nil {
		t.Fatalf("SeasonAt(%v): unexpected error: %v", when, err)
	}
	return info
}

func TestSeasonAtKnownDates(t *testing.T) {
	cases := []struct {
		name  string
		when  time.Time
		north earth.Season
		south earth.Season
	}{
		{"late June", time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC), earth.Summer, earth.Winter},
		{"late December", time.Date(2026, 12, 25, 12, 0, 0, 0, time.UTC), earth.Winter, earth.Summer},
		{"late March", time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC), earth.Spring, earth.Fall},
		{"late September", time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), earth.Fall, earth.Spring},
		{"mid January", time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), earth.Winter, earth.Summer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := seasonAt(t, northObs, tc.when).Season; got != tc.north {
				t.Errorf("north: got %v, want %v", got, tc.north)
			}
			if got := seasonAt(t, southObs, tc.when).Season; got != tc.south {
				t.Errorf("south: got %v, want %v", got, tc.south)
			}
		})
	}
}

func TestSeasonAtHemisphereFlip(t *testing.T) {
	// The same instant is a different season in each hemisphere, and the two
	// are always six months (two seasons) apart.
	for _, when := range []time.Time{
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
	} {
		n := seasonAt(t, northObs, when).Season
		s := seasonAt(t, southObs, when).Season
		if n == s {
			t.Errorf("%v: north and south seasons must differ, both %v", when, n)
		}
		if (int(n)+2)%4 != int(s) {
			t.Errorf("%v: south %v is not the opposite of north %v", when, s, n)
		}
	}
}

func TestSeasonAtNextSeason(t *testing.T) {
	info := seasonAt(t, northObs, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if info.Season != earth.Summer {
		t.Fatalf("season: got %v, want Summer", info.Season)
	}
	if info.Next != earth.Fall {
		t.Errorf("next: got %v, want Fall", info.Next)
	}
}

func TestSeasonAtProgressRangeAndMonotonic(t *testing.T) {
	// Sample through Northern-hemisphere summer 2026 and confirm progress rises
	// strictly through [0, 1) while the season stays Summer.
	samples := []time.Time{
		time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
	}
	prev := -1.0
	for _, when := range samples {
		info := seasonAt(t, northObs, when)
		if info.Season != earth.Summer {
			t.Fatalf("%v: expected Summer, got %v", when, info.Season)
		}
		if info.Progress < 0 || info.Progress >= 1 {
			t.Errorf("%v: progress %v out of [0,1)", when, info.Progress)
		}
		if info.Progress <= prev {
			t.Errorf("%v: progress %v not greater than previous %v", when, info.Progress, prev)
		}
		prev = info.Progress
	}
}

func TestSeasonAtDayCountsAndBounds(t *testing.T) {
	info := seasonAt(t, northObs, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if !info.Start.Before(info.End) {
		t.Fatalf("start %v must be before end %v", info.Start, info.End)
	}
	if info.DaysElapsed < 0 {
		t.Errorf("DaysElapsed negative: %v", info.DaysElapsed)
	}
	if info.DaysUntilNext <= 0 {
		t.Errorf("DaysUntilNext must be positive: %v", info.DaysUntilNext)
	}
	// An astronomical season is about three months long.
	total := info.DaysElapsed + info.DaysUntilNext
	if total < 85 || total > 95 {
		t.Errorf("season length %v days outside expected 85..95", total)
	}
	// Progress agrees with the day counts.
	wantProg := info.DaysElapsed / total
	if diff := info.Progress - wantProg; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("progress %v disagrees with days ratio %v", info.Progress, wantProg)
	}
}

func TestSeasonAtBoundaryTransition(t *testing.T) {
	// Straddle the June solstice: the instant just before belongs to the
	// previous season and its End equals the next season's Start, so a live
	// clock crosses the boundary with no gap.
	info := seasonAt(t, northObs, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	boundary := info.Start

	before := seasonAt(t, northObs, boundary.Add(-time.Hour))
	after := seasonAt(t, northObs, boundary.Add(time.Hour))

	if before.Season != earth.Spring {
		t.Errorf("before boundary: got %v, want Spring", before.Season)
	}
	if after.Season != earth.Summer {
		t.Errorf("after boundary: got %v, want Summer", after.Season)
	}
	if !before.End.Equal(after.Start) {
		t.Errorf("boundary not continuous: before.End %v != after.Start %v", before.End, after.Start)
	}
	if before.Next != after.Season {
		t.Errorf("before.Next %v should equal after.Season %v", before.Next, after.Season)
	}
}

func TestSeasonAtInvalidObserver(t *testing.T) {
	when := time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)

	_, err := earth.SeasonAt(astronomy.Observer{Lat: 91, Lng: 0}, when)
	if !errors.Is(err, earth.ErrInvalidLatitude) {
		t.Errorf("latitude: got %v, want ErrInvalidLatitude", err)
	}
	if code, ok := apperr.Code(err); !ok || code != astronomy.CodeInvalidLatitude {
		t.Errorf("latitude code: got %d, want %d", code, astronomy.CodeInvalidLatitude)
	}

	_, err = earth.SeasonAt(astronomy.Observer{Lat: 0, Lng: 181}, when)
	if !errors.Is(err, earth.ErrInvalidLongitude) {
		t.Errorf("longitude: got %v, want ErrInvalidLongitude", err)
	}
	if code, ok := apperr.Code(err); !ok || code != astronomy.CodeInvalidLongitude {
		t.Errorf("longitude code: got %d, want %d", code, astronomy.CodeInvalidLongitude)
	}
}

func TestSeasonString(t *testing.T) {
	cases := map[earth.Season]string{
		earth.Spring: "spring",
		earth.Summer: "summer",
		earth.Fall:   "fall",
		earth.Winter: "winter",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("Season(%d).String() = %q, want %q", int(s), got, want)
		}
	}
	if got := earth.Season(99).String(); got != "unknown" {
		t.Errorf("out-of-range Season.String() = %q, want %q", got, "unknown")
	}
}
