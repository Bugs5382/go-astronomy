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
	"github.com/soniakeys/meeus/v3/coord"
	"github.com/soniakeys/unit"

	"github.com/Bugs5382/go-astronomy/internal/angles"
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
	obl := coord.NewObliquity(unit.AngleFromDeg(obliquityDeg))
	lon, lat := coord.EqToEcl(unit.RAFromDeg(eq.RA), unit.AngleFromDeg(eq.Dec), obl.S, obl.C)
	return Ecliptic{Lon: angles.Normalize(lon.Deg()), Lat: lat.Deg()}
}

// EclipticToEquatorial converts an ecliptic position to equatorial coordinates
// using the given obliquity of the ecliptic in degrees.
func EclipticToEquatorial(ecl Ecliptic, obliquityDeg float64) Equatorial {
	obl := coord.NewObliquity(unit.AngleFromDeg(obliquityDeg))
	ra, dec := coord.EclToEq(unit.AngleFromDeg(ecl.Lon), unit.AngleFromDeg(ecl.Lat), obl.S, obl.C)
	return Equatorial{RA: angles.Normalize(ra.Deg()), Dec: dec.Deg()}
}

// EquatorialToHorizontal converts an equatorial position to horizontal
// coordinates for an observer at the given latitude and east-positive longitude
// (degrees), with gstDeg the Greenwich sidereal time in degrees. The sidereal
// time must be consistent with the equatorial coordinates (mean with mean,
// apparent with apparent).
func EquatorialToHorizontal(eq Equatorial, latDeg, lonEastDeg, gstDeg float64) Horizontal {
	st := unit.TimeFromHour(gstDeg / 15)
	// meeus uses west-positive observer longitude and returns azimuth measured
	// westward from the south; negate longitude and rotate the azimuth by 180
	// degrees to reach clockwise-from-north.
	a, h := coord.EqToHz(
		unit.RAFromDeg(eq.RA),
		unit.AngleFromDeg(eq.Dec),
		unit.AngleFromDeg(latDeg),
		unit.AngleFromDeg(-lonEastDeg),
		st,
	)
	return Horizontal{
		Azimuth:  angles.Normalize(a.Deg() + 180),
		Altitude: h.Deg(),
	}
}

// HorizontalToEquatorial converts a horizontal position back to equatorial
// coordinates for an observer at the given latitude and east-positive longitude
// (degrees), with gstDeg the Greenwich sidereal time in degrees. It is the
// inverse of EquatorialToHorizontal.
func HorizontalToEquatorial(hz Horizontal, latDeg, lonEastDeg, gstDeg float64) Equatorial {
	st := unit.TimeFromHour(gstDeg / 15)
	a := unit.AngleFromDeg(hz.Azimuth - 180) // back to westward-from-south
	ra, dec := coord.HzToEq(
		a,
		unit.AngleFromDeg(hz.Altitude),
		unit.AngleFromDeg(latDeg),
		unit.AngleFromDeg(-lonEastDeg),
		st,
	)
	return Equatorial{RA: angles.Normalize(ra.Deg()), Dec: dec.Deg()}
}
