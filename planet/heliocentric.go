package planet

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

	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// HeliocentricPosition is a planet's place as seen from the centre of the Sun,
// referred to the ecliptic and equinox of date (the dynamical frame of
// VSOP87D). It is geometric: no light-time or aberration, because it belongs
// to no observer.
type HeliocentricPosition struct {
	// Lon is the ecliptic longitude in degrees, [0, 360).
	Lon float64
	// Lat is the ecliptic latitude in degrees.
	Lat float64
	// DistanceAU is the distance from the Sun in astronomical units.
	DistanceAU float64
}

// Vector returns the position as rectangular ecliptic coordinates in
// astronomical units: x toward the equinox of date, z toward the north
// ecliptic pole. The view of one planet from another is the difference of
// their vectors at the same instant (before light-time).
func (h HeliocentricPosition) Vector() [3]float64 {
	l, b := h.Lon*math.Pi/180, h.Lat*math.Pi/180
	return [3]float64{
		h.DistanceAU * math.Cos(b) * math.Cos(l),
		h.DistanceAU * math.Cos(b) * math.Sin(l),
		h.DistanceAU * math.Sin(b),
	}
}

// EarthHeliocentric returns Earth's heliocentric position at instant t, for
// differencing with another planet's. It uses the Earth table the Sun's
// position already needs, so it links no planet table.
func EarthHeliocentric(t time.Time) HeliocentricPosition {
	l, b, r := ephemeris.EarthHeliocentric(julian.TT(t))
	return HeliocentricPosition{Lon: l, Lat: b, DistanceAU: r}
}
