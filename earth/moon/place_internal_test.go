package moon

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

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// TestInterpolatedMoonMatchesDirect checks the hourly interpolation the rise
// and set searches use against the direct Moon place, over five years at
// off-hour instants (every phase of the orbit, perigee and apogee, and the RA
// wrap): under 1e-7 degrees (it measures 3.5e-8, a ten-thousandth of an arc
// second), and exact at a whole hour.
func TestInterpolatedMoonMatchesDirect(t *testing.T) {
	t.Parallel()
	places := newMoonPlaces()
	var worst float64
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for m := 0; m < 5*365*24*60; m += 613 {
		when := start.Add(time.Duration(m)*time.Minute + 17*time.Second + 300*time.Millisecond)
		d, i := moonPlaceAt(when), places.at(when)
		dra := math.Mod(d.ra-i.ra+540, 360) - 180
		worst = math.Max(worst, math.Max(math.Abs(dra), math.Abs(d.dec-i.dec)))
	}
	if worst > 1e-7 {
		t.Errorf("interpolated Moon off the direct place by %.2e degrees", worst)
	}
	hour := time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC)
	if places.at(hour) != moonPlaceAt(hour) {
		t.Error("a whole hour is not the direct place")
	}
}

// TestInterpolatedRiseSetMatchesDirect checks moonrise and moonset through
// the interpolated Moon against the same search on direct places: within the
// one second the bisection resolves, over two months at three latitudes.
func TestInterpolatedRiseSetMatchesDirect(t *testing.T) {
	t.Parallel()
	for _, obs := range []astronomy.Observer{
		{Lat: 40.678, Lng: -73.944},
		{Lat: 39.74, Lng: -104.99, Height: astronomy.Meters(1609)},
		{Lat: -33.9, Lng: 151.2},
	} {
		dip := earth.HorizonDip(obs.Height)
		direct := func(when time.Time) float64 {
			hz, semi := topocentric(obs, when)
			return hz.Altitude + horizonRefraction + semi + dip
		}
		for d := 0; d < 60; d += 3 {
			from := time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC).AddDate(0, 0, d)
			for _, rising := range []bool{true, false} {
				got, ok, err := nextCrossing(obs, from, rising)
				if err != nil || !ok {
					t.Fatalf("nextCrossing: %v %v", ok, err)
				}
				want := directCrossing(direct, from, rising)
				if dt := got.Sub(want).Abs(); dt > time.Second {
					t.Errorf("%v %s rising=%v: %s, direct %s (off %v)", obs.Lat, from.Format(time.DateOnly), rising, got, want, dt)
				}
			}
		}
	}
}

// directCrossing is nextCrossing's scan on direct places.
func directCrossing(f func(time.Time) float64, t time.Time, rising bool) time.Time {
	prev := f(t)
	for when := t.Add(riseSetSearchStep); ; when = when.Add(riseSetSearchStep) {
		cur := f(when)
		if (rising && prev < 0 && cur >= 0) || (!rising && prev >= 0 && cur < 0) {
			return bisectCrossing(f, when.Add(-riseSetSearchStep), when)
		}
		prev = cur
	}
}
