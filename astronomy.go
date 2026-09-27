package astronomy

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

import "time"

// Observer is a location on the Earth's surface for which celestial positions
// are computed. Time is never captured here; it is always supplied as a
// parameter to the calculation functions, keeping the library stateless.
type Observer struct {
	// Lat is the geographic latitude in degrees, positive north, in [-90, 90].
	Lat float64
	// Lng is the geographic longitude in degrees, positive east, in [-180, 180].
	Lng float64
	// TZ is the observer's local time zone. A nil value is treated as UTC.
	TZ *time.Location
	// Height is the observer's height above sea level, optional: the zero
	// value is SeaLevel, which reproduces the sea-level answers exactly and
	// needs no lookup. Set it with Meters or Feet, or look it up with an
	// ElevationResolver (see ResolveObserverWith).
	//
	// From height the sea horizon lies below the astronomical horizon by the
	// dip (see earth.HorizonDip), so the Sun and Moon rise earlier and set
	// later; the height also enters their parallax. The dip assumes an
	// unobstructed sea horizon. For an observer in motion, such as a plane,
	// pass the position and height for each instant to each call.
	Height Height
}

// Location returns the observer's time zone, defaulting to UTC when TZ is nil.
func (o Observer) Location() *time.Location {
	if o.TZ == nil {
		return time.UTC
	}
	return o.TZ
}

// Horizontal is a direction in the local horizontal (alt-az) frame. Azimuth is
// measured clockwise from true north and Altitude is the angle above the
// horizon; both are in degrees. A negative Altitude is below the horizon.
type Horizontal struct {
	Altitude float64
	Azimuth  float64
}

// AboveHorizon reports whether the direction is above the geometric horizon.
func (h Horizontal) AboveHorizon() bool { return h.Altitude > 0 }

// AngularDiameter is the apparent angular size of a body's disc, in degrees.
type AngularDiameter float64

// Radius returns half the angular diameter, in degrees.
func (a AngularDiameter) Radius() float64 { return float64(a) / 2 }

// Position is the apparent placement of a body: the direction to the center of
// its disc paired with the disc's apparent angular size. Pairing the center
// with the angular diameter lets a consumer size and place the disc and reason
// about alignment and overlap between bodies.
type Position struct {
	Horizontal
	Diameter AngularDiameter
}
