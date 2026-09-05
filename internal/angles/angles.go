// Package angles provides small, dependency-free helpers for working with
// angles expressed in degrees: conversion to and from radians, normalization
// into the range [0, 360), and clamping of declinations to [-90, 90].
package angles

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

import "math"

const (
	radPerDeg = math.Pi / 180
	degPerRad = 180 / math.Pi
)

// DegToRad converts an angle in degrees to radians.
func DegToRad(deg float64) float64 { return deg * radPerDeg }

// RadToDeg converts an angle in radians to degrees.
func RadToDeg(rad float64) float64 { return rad * degPerRad }

// Normalize reduces an angle in degrees to the half-open range [0, 360).
func Normalize(deg float64) float64 {
	d := math.Mod(deg, 360)
	if d < 0 {
		d += 360
	}
	// math.Mod can leave a value of exactly 360 after the adjustment above only
	// through rounding of a tiny negative input; guard the invariant explicitly.
	if d >= 360 {
		d -= 360
	}
	return d
}

// ClampDeclination constrains a declination in degrees to the physical range
// [-90, 90], the interval spanned by the celestial poles.
func ClampDeclination(dec float64) float64 {
	switch {
	case dec > 90:
		return 90
	case dec < -90:
		return -90
	default:
		return dec
	}
}
