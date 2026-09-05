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

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// TestTrackInvariants checks the sample count, inclusive endpoints, monotonic
// non-decreasing time, the [0,1] progress span, the azimuth range, and that
// illumination and phase are populated on every sample.
func TestTrackInvariants(t *testing.T) {
	t.Parallel()
	obs := brooklyn()
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	const n = 25

	samples, err := moon.Track(obs, from, to, n)
	if err != nil {
		t.Fatalf("Track error: %v", err)
	}
	if len(samples) != n {
		t.Fatalf("len(samples) = %d, want %d", len(samples), n)
	}
	if !samples[0].Time.Equal(from) {
		t.Errorf("first sample %s, want %s", samples[0].Time, from)
	}
	if !samples[n-1].Time.Equal(to) {
		t.Errorf("last sample %s, want %s", samples[n-1].Time, to)
	}
	if samples[0].TimeProgress != 0 || samples[n-1].TimeProgress != 1 {
		t.Errorf("progress endpoints = %v..%v, want 0..1", samples[0].TimeProgress, samples[n-1].TimeProgress)
	}
	for i, s := range samples {
		if s.Azimuth < 0 || s.Azimuth >= 360 {
			t.Errorf("sample %d azimuth %.4f out of [0,360)", i, s.Azimuth)
		}
		if s.Illumination < 0 || s.Illumination > 1 {
			t.Errorf("sample %d illumination %.4f out of [0,1]", i, s.Illumination)
		}
		if s.Phase < moon.New || s.Phase > moon.WaningCrescent {
			t.Errorf("sample %d phase %v out of range", i, s.Phase)
		}
		if i > 0 && !s.Time.After(samples[i-1].Time) {
			t.Errorf("sample %d time %s not after previous %s", i, s.Time, samples[i-1].Time)
		}
	}
}

// TestTrackTooFewSamples returns nil for fewer than two samples, which cannot
// define a progress span.
func TestTrackTooFewSamples(t *testing.T) {
	t.Parallel()
	obs := brooklyn()
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, n := range []int{-1, 0, 1} {
		got, err := moon.Track(obs, from, from.Add(time.Hour), n)
		if err != nil {
			t.Errorf("Track(samples=%d) error: %v", n, err)
		}
		if got != nil {
			t.Errorf("Track(samples=%d) = %v, want nil", n, got)
		}
	}
}

// TestTrackInvalidObserver verifies Track rejects an out-of-range observer.
func TestTrackInvalidObserver(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	bad := astronomy.Observer{Lat: 0, Lng: 200}
	if _, err := moon.Track(bad, from, from.Add(time.Hour), 10); !errors.Is(err, moon.ErrInvalidLongitude) {
		t.Errorf("Track err = %v, want invalid longitude", err)
	}
}
