package earth

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// sunPlaces serves the Sun's apparent geocentric place at any instant from
// the places at whole UTC hours, by four-point (cubic) Lagrange
// interpolation, computing each hour once. Resolving a day samples the Sun
// about 1500 times, and VSOP87 with the nutation is two thirds of a Sun
// position; the place moves smoothly over hours (about 0.04 degrees an hour
// in right ascension), so 30 hourly places carry the whole day. The sidereal
// time, the parallax, and the horizon are still computed at each instant.
//
// The interpolated place differs from the direct one by under 1e-9 degrees
// (TestInterpolatedSunMatchesDirect), a few millionths of an arc second, and
// is exact at whole hours; before 1972, where TT - UTC steps at each midnight,
// by under 5e-9. A day's band boundaries move by under 0.1 s, inside the
// 0.2 s the crossing search resolves (TestInterpolatedDayMatchesDirect). A
// sunPlaces is for one goroutine.
type sunPlaces struct {
	nodes map[int64]sunPlace
}

func newSunPlaces() *sunPlaces {
	return &sunPlaces{nodes: make(map[int64]sunPlace, 32)}
}

// node returns the Sun's place at the whole UTC hour h (hours since the Unix
// epoch).
func (c *sunPlaces) node(h int64) sunPlace {
	if p, ok := c.nodes[h]; ok {
		return p
	}
	p := sunPlaceAt(time.Unix(h*3600, 0))
	c.nodes[h] = p
	return p
}

// at returns the Sun's place at t, interpolated between the hours around it.
func (c *sunPlaces) at(t time.Time) sunPlace {
	sec := t.Unix()
	h := sec / 3600
	if sec%3600 < 0 {
		h-- // floor for instants before 1970
	}
	x := (float64(sec-h*3600) + float64(t.Nanosecond())/1e9) / 3600 // [0, 1)
	p0, p1, p2, p3 := c.node(h-1), c.node(h), c.node(h+1), c.node(h+2)

	// Lagrange weights for the nodes at -1, 1, and 2 hours. The weights sum to
	// one, so the value is the node at 0 plus the weighted steps to the
	// others, and at x = 0, where these weights are zero, a whole hour gives
	// the node's place exactly.
	w0 := -x * (x - 1) * (x - 2) / 6
	w2 := -(x + 1) * x * (x - 2) / 2
	w3 := (x + 1) * x * (x - 1) / 6
	interp := func(v0, v1, v2, v3 float64) float64 {
		return v1 + (w0*(v0-v1) + w2*(v2-v1) + w3*(v3-v1))
	}
	// Right ascension wraps at 360; interpolate the steps from the node at 0.
	unwrap := func(v float64) float64 {
		d := v - p1.ra
		switch {
		case d > 180:
			d -= 360
		case d < -180:
			d += 360
		}
		return d
	}
	ra := p1.ra + (w0*unwrap(p0.ra) + w2*unwrap(p2.ra) + w3*unwrap(p3.ra))
	ra = math.Mod(ra, 360)
	if ra < 0 {
		ra += 360
	}
	return sunPlace{
		ra:          ra,
		dec:         interp(p0.dec, p1.dec, p2.dec, p3.dec),
		distKm:      interp(p0.distKm, p1.distKm, p2.distKm, p3.distKm),
		eqEquinoxes: interp(p0.eqEquinoxes, p1.eqEquinoxes, p2.eqEquinoxes, p3.eqEquinoxes),
	}
}

// sunAltitudes returns the Sun's altitude for the observer as a function of
// the instant, for the searches that resolve a day. The Sun's place comes
// from hourly places (sunPlaces), and every altitude is remembered, because
// the day's four scans (the level crossings, solar noon, the day's minimum,
// and the lower culminations) walk the same one-minute grid. The function is
// for one goroutine; each constructor call makes its own.
func sunAltitudes(obs astronomy.Observer) func(time.Time) float64 {
	type instant struct {
		sec  int64
		nsec int
	}
	places := newSunPlaces()
	cache := make(map[instant]float64, 4096)
	return func(t time.Time) float64 {
		k := instant{t.Unix(), t.Nanosecond()}
		if a, ok := cache[k]; ok {
			return a
		}
		a := sunHorizontal(obs, t, places.at(t)).Altitude
		cache[k] = a
		return a
	}
}
