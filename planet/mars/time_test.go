package mars_test

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
	"math"
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/planet/mars"
)

// TestSolDate checks the Mars Sol Date against the worked value of Allison
// and McEwen (2000) as used by Mars24: 44795.9998 at 2000 January 6, 00:00
// UTC, and one sol later exactly one more.
func TestSolDate(t *testing.T) {
	t.Parallel()
	epoch := time.Date(2000, 1, 6, 0, 0, 0, 0, time.UTC)
	if got := mars.SolDate(epoch); math.Abs(got-44795.9998) > 2e-4 {
		t.Errorf("SolDate(2000-01-06) = %.5f, want 44795.9998", got)
	}
	if d := mars.SolDate(epoch.Add(mars.Sol)) - mars.SolDate(epoch); math.Abs(d-1) > 1e-9 {
		t.Errorf("a Sol later the date moved by %.10f", d)
	}
	if math.Abs(mars.Sol.Seconds()-1.0274912517*86400) > 1e-6 {
		t.Errorf("Sol = %v", mars.Sol)
	}
}

// TestLocalMeanSolarTime checks the local mean solar time follows the
// longitude at 15 degrees a Mars hour and stays in [0, 24).
func TestLocalMeanSolarTime(t *testing.T) {
	t.Parallel()
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	mtc := mars.LocalMeanSolarTime(when, 0)
	if want := math.Mod(24*mars.SolDate(when), 24); math.Abs(mtc-want) > 1e-9 {
		t.Errorf("prime meridian %.6f, want %.6f", mtc, want)
	}
	for _, lon := range []float64{-180, -77.45, 15, 77.45, 180} {
		got := mars.LocalMeanSolarTime(when, lon)
		want := math.Mod(mtc+lon/15+48, 24)
		if got < 0 || got >= 24 || math.Abs(got-want) > 1e-9 {
			t.Errorf("lon %v: %.6f, want %.6f", lon, got, want)
		}
	}
}
