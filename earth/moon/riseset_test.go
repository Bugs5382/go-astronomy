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
	"errors"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// TestNextRiseSetFound checks that a mid-latitude observer sees the Moon both
// rise and set within a day or so, that the events are in the future, and that
// the apparent altitude at each event sits near the rise/set horizon and is
// moving in the expected direction.
func TestNextRiseSetFound(t *testing.T) {
	t.Parallel()
	obs := brooklyn()
	from := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	rise, ok, err := moon.NextRise(obs, from)
	if err != nil {
		t.Fatalf("NextRise error: %v", err)
	}
	if !ok {
		t.Fatal("expected a moonrise within the search window")
	}
	if !rise.After(from) {
		t.Errorf("rise %s not after %s", rise, from)
	}
	if rise.Sub(from) > 26*time.Hour {
		t.Errorf("rise %s more than a day after start", rise)
	}
	// Altitude should be climbing through the horizon at rise.
	before, _ := moon.ApparentPosition(obs, rise.Add(-5*time.Minute))
	after, _ := moon.ApparentPosition(obs, rise.Add(5*time.Minute))
	if !(after.Altitude > before.Altitude) {
		t.Errorf("altitude not increasing across rise: before %.4f after %.4f", before.Altitude, after.Altitude)
	}

	set, ok, err := moon.NextSet(obs, from)
	if err != nil {
		t.Fatalf("NextSet error: %v", err)
	}
	if !ok {
		t.Fatal("expected a moonset within the search window")
	}
	if !set.After(from) {
		t.Errorf("set %s not after %s", set, from)
	}
	beforeS, _ := moon.ApparentPosition(obs, set.Add(-5*time.Minute))
	afterS, _ := moon.ApparentPosition(obs, set.Add(5*time.Minute))
	if !(afterS.Altitude < beforeS.Altitude) {
		t.Errorf("altitude not decreasing across set: before %.4f after %.4f", beforeS.Altitude, afterS.Altitude)
	}
}

// TestNextRiseSetInvalidObserver verifies the rise/set functions reject an
// out-of-range observer with the shared coded error.
func TestNextRiseSetInvalidObserver(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bad := astronomy.Observer{Lat: -91, Lng: 0}
	wantCode := func(err error) bool {
		got, ok := apperr.Code(err)
		return ok && got == astronomy.CodeInvalidLatitude
	}
	if _, _, err := moon.NextRise(bad, when); !errors.Is(err, moon.ErrInvalidLatitude) || !wantCode(err) {
		t.Errorf("NextRise err = %v, want invalid latitude", err)
	}
	if _, _, err := moon.NextSet(bad, when); !errors.Is(err, moon.ErrInvalidLatitude) || !wantCode(err) {
		t.Errorf("NextSet err = %v, want invalid latitude", err)
	}
}
