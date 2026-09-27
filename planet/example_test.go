package planet_test

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
	"fmt"
	"math"
	"time"

	"github.com/Bugs5382/go-astronomy/planet"
	"github.com/Bugs5382/go-astronomy/planet/mars"
)

// Earth's heliocentric position, which every planet's view from Earth starts
// from.
func ExampleEarthHeliocentric() {
	e := planet.EarthHeliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	fmt.Printf("L %.4f B %.4f R %.6f au\n", e.Lon, e.Lat, e.DistanceAU)
	// Output:
	// L 100.3232 B 0.0001 R 0.983343 au
}

// The geometric view of Mars from Earth is the difference of two heliocentric
// vectors at the same instant (no light-time or aberration).
func ExampleHeliocentricPosition_Vector() {
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	ev, mv := planet.EarthHeliocentric(when).Vector(), mars.Heliocentric(when).Vector()
	d := [3]float64{mv[0] - ev[0], mv[1] - ev[1], mv[2] - ev[2]}
	dist := math.Sqrt(d[0]*d[0] + d[1]*d[1] + d[2]*d[2])
	lon := math.Mod(math.Atan2(d[1], d[0])*180/math.Pi+360, 360)
	fmt.Printf("Mars from Earth: %.4f au, ecliptic longitude %.2f degrees\n", dist, lon)
	// Output:
	// Mars from Earth: 0.9139 au, ecliptic longitude 159.87 degrees
}
