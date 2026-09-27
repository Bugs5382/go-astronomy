package satellite

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

// Constants shared by the propagator and the frame conversions.

import (
	"math"

	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

const (
	// earthRotationRadS is the Earth's rotation rate used with SGP4's TEME
	// frame (Vallado 2004).
	earthRotationRadS = 7.292115e-5
	// wgs84A and wgs84F give the observer's place on the WGS84 ellipsoid.
	wgs84A = ephemeris.EarthEquatorialRadiusKm
	wgs84F = ephemeris.EarthFlattening
)

const (
	twoPi  = 2.0 * math.Pi
	deg2rd = math.Pi / 180.0
	x2o3   = 2.0 / 3.0

	// xpdotp converts revolutions per day to radians per minute.
	xpdotp = 1440.0 / (2.0 * math.Pi)

	// jd1950 is the Julian date of 0 January 1950, 0h, the SGP4 epoch
	// origin.
	jd1950 = 2433281.5
)

// WGS-72 gravity constants, as returned by getgravconst("wgs72").
var (
	wgs72Mu            = 398600.8 // km^3/s^2
	wgs72RadiusEarthKm = 6378.135 // km
	wgs72Xke           = 60.0 / math.Sqrt(wgs72RadiusEarthKm*wgs72RadiusEarthKm*wgs72RadiusEarthKm/wgs72Mu)
	wgs72J2            = 0.001082616
	wgs72J3            = -0.00000253881
	wgs72J4            = -0.00000165597
	wgs72J3oJ2         = wgs72J3 / wgs72J2
)
