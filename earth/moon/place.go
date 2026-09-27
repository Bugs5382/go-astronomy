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
	"time"
)

// moonPlaces serves the Moon's apparent geocentric place at any instant from
// its places at whole UTC hours, by four-point (cubic) Lagrange
// interpolation, computing each hour once. The ELP 2000-82B series is most of
// the cost of a Moon position, and the geocentric place moves smoothly over
// hours (about half a degree an hour), so a rise or set search that samples
// the Moon every ten minutes and bisects needs the series only at the hours it
// passes. The sidereal time and the parallax are still computed at each
// instant.
//
// The interpolated place differs from the direct one by under 1e-7 degrees
// (it measures 3.5e-8, a ten-thousandth of an arc second;
// TestInterpolatedMoonMatchesDirect) and is exact at whole hours, and a rise
// or set found through it is within the search's one-second resolution of
// the direct one. A moonPlaces is for one goroutine.
type moonPlaces struct {
	nodes map[int64]moonPlace
}

func newMoonPlaces() *moonPlaces {
	return &moonPlaces{nodes: make(map[int64]moonPlace, 32)}
}

// node returns the Moon's place at the whole UTC hour h (hours since the Unix
// epoch).
func (c *moonPlaces) node(h int64) moonPlace {
	if p, ok := c.nodes[h]; ok {
		return p
	}
	p := moonPlaceAt(time.Unix(h*3600, 0))
	c.nodes[h] = p
	return p
}

// at returns the Moon's place at t, interpolated between the hours around it.
func (c *moonPlaces) at(t time.Time) moonPlace {
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
	return moonPlace{
		ra:          ra,
		dec:         interp(p0.dec, p1.dec, p2.dec, p3.dec),
		distKm:      interp(p0.distKm, p1.distKm, p2.distKm, p3.distKm),
		eqEquinoxes: interp(p0.eqEquinoxes, p1.eqEquinoxes, p2.eqEquinoxes, p3.eqEquinoxes),
	}
}
