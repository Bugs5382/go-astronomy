// Package coordinates converts celestial positions between the ecliptic,
// equatorial, and local horizontal (alt-az) frames. It wraps the meeus coord
// transforms behind a small API that speaks plain float64 degrees and uses the
// conventions the rest of the library expects: east-positive observer
// longitude and azimuth measured clockwise from true north.
package coordinates

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

	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
)

// Equatorial is a position in the equatorial frame. RA is right ascension and
// Dec is declination, both in degrees.
type Equatorial struct {
	RA  float64
	Dec float64
}

// Ecliptic is a position in the ecliptic frame. Lon is ecliptic longitude and
// Lat is ecliptic latitude, both in degrees.
type Ecliptic struct {
	Lon float64
	Lat float64
}

// Horizontal is a position in the local horizontal frame. Azimuth is measured
// clockwise from true north and Altitude is the angle above the horizon, both
// in degrees.
type Horizontal struct {
	Azimuth  float64
	Altitude float64
}

// EquatorialToEcliptic converts an equatorial position to ecliptic coordinates
// using the given obliquity of the ecliptic in degrees.
func EquatorialToEcliptic(eq Equatorial, obliquityDeg float64) Ecliptic {
	lon, lat := ephemeris.EqToEcl(eq.RA, eq.Dec, obliquityDeg)
	return Ecliptic{Lon: angles.Normalize(lon), Lat: lat}
}

// EclipticToEquatorial converts an ecliptic position to equatorial coordinates
// using the given obliquity of the ecliptic in degrees.
func EclipticToEquatorial(ecl Ecliptic, obliquityDeg float64) Equatorial {
	ra, dec := ephemeris.EclToEq(ecl.Lon, ecl.Lat, obliquityDeg)
	return Equatorial{RA: angles.Normalize(ra), Dec: dec}
}

// PrecessEquatorial precesses an equatorial position from the epochFrom equinox
// to the epochTo equinox, both given as Julian years (for example 2000.0 for
// J2000.0). Proper motion is not applied; the transform accounts for the
// precession of the equinoxes only. It uses the rigorous precession model of
// Meeus chapter 21 and returns coordinates with RA normalized to [0, 360).
func PrecessEquatorial(eq Equatorial, epochFrom, epochTo float64) Equatorial {
	ra, dec := ephemeris.PrecessEq(eq.RA, eq.Dec, epochFrom, epochTo)
	return Equatorial{RA: angles.Normalize(ra), Dec: dec}
}

// EquatorialToHorizontal converts an equatorial position to horizontal
// coordinates for an observer at the given latitude and east-positive longitude
// (degrees), with gstDeg the Greenwich sidereal time in degrees. The sidereal
// time must be consistent with the equatorial coordinates (mean with mean,
// apparent with apparent).
func EquatorialToHorizontal(eq Equatorial, latDeg, lonEastDeg, gstDeg float64) Horizontal {
	// The book's transform takes west-positive observer longitude and returns
	// azimuth measured westward from the south; negate the longitude and rotate
	// the azimuth by 180 degrees to reach clockwise-from-north.
	az, alt := ephemeris.EqToHz(eq.RA, eq.Dec, latDeg, -lonEastDeg, gstDeg)
	return Horizontal{
		Azimuth:  angles.Normalize(az + 180),
		Altitude: alt,
	}
}

// HorizontalToEquatorial converts a horizontal position back to equatorial
// coordinates for an observer at the given latitude and east-positive longitude
// (degrees), with gstDeg the Greenwich sidereal time in degrees. It is the
// inverse of EquatorialToHorizontal.
func HorizontalToEquatorial(hz Horizontal, latDeg, lonEastDeg, gstDeg float64) Equatorial {
	// Rotate the azimuth back to westward-from-south and negate the longitude.
	ra, dec := ephemeris.HzToEq(hz.Azimuth-180, hz.Altitude, latDeg, -lonEastDeg, gstDeg)
	return Equatorial{RA: angles.Normalize(ra), Dec: dec}
}

// PositionAngle returns the position angle, in degrees in [0, 360), of the
// direction from the point (raDeg, decDeg) toward the point (ra0Deg,
// dec0Deg): measured at the first point from the direction of the north
// celestial pole, turning through east (Meeus 48.5, where it gives the bright
// limb of the Moon from the Sun's place). Any spherical frame works: in the
// horizontal frame, pass the negated azimuths as right ascensions and the
// altitudes as declinations, and the angle is measured from the zenith.
func PositionAngle(ra0Deg, dec0Deg, raDeg, decDeg float64) float64 {
	d0 := angles.DegToRad(dec0Deg)
	d := angles.DegToRad(decDeg)
	dra := angles.DegToRad(ra0Deg - raDeg)
	y := math.Cos(d0) * math.Sin(dra)
	x := math.Sin(d0)*math.Cos(d) - math.Cos(d0)*math.Sin(d)*math.Cos(dra)
	return angles.Normalize(angles.RadToDeg(math.Atan2(y, x)))
}
