package horizons

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
	"context"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Geocentric is a target's geocentric apparent place: right ascension and
// declination of date, in degrees, and the distance from the Earth's centre,
// in km, as Horizons computes it (light-time, aberration, and nutation
// included).
type Geocentric struct {
	RA, Dec    float64
	DistanceKm float64
}

// Position is a target as an observer sees it.
type Position struct {
	// Horizontal is the airless (unrefracted) altitude and azimuth, in
	// degrees, azimuth clockwise from true north.
	astronomy.Horizontal
	// RA and Dec are the topocentric apparent right ascension and
	// declination of date, in degrees.
	RA, Dec float64
	// RangeKm is the distance from the observer.
	RangeKm float64
}

// Geocentric returns the target's geocentric apparent place at t,
// interpolated from the cached window, fetching the window on a miss. The
// target is a Horizons command such as "-170" (the James Webb Space
// Telescope).
func (c *Client) Geocentric(ctx context.Context, target string, t time.Time) (Geocentric, error) {
	if err := ctx.Err(); err != nil {
		return Geocentric{}, err
	}
	tab, err := c.tableFor(ctx, target, t)
	if err != nil {
		return Geocentric{}, err
	}
	s, err := tab.at(t)
	if err != nil {
		return Geocentric{}, err
	}
	return Geocentric{RA: s.RA, Dec: s.Dec, DistanceKm: s.Km}, nil
}

// Position returns the target as the observer sees it at t: the geocentric
// place moved to the observer on the WGS84 ellipsoid by the rigorous parallax
// correction (Meeus chapter 40), then to altitude and azimuth with apparent
// sidereal time. Only the geocentric window is fetched, so every observer
// shares it.
func (c *Client) Position(ctx context.Context, target string, obs astronomy.Observer, t time.Time) (Position, error) {
	if obs.Lat < -90 || obs.Lat > 90 {
		return Position{}, apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if obs.Lng < -180 || obs.Lng > 180 {
		return Position{}, apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	g, err := c.Geocentric(ctx, target, t)
	if err != nil {
		return Position{}, err
	}
	gst := julian.ApparentSiderealTime(t)
	ra, dec, km := ephemeris.Topocentric(g.RA, g.Dec, g.DistanceKm, obs.Lat, 0, gst+obs.Lng)
	hz := coordinates.EquatorialToHorizontal(coordinates.Equatorial{RA: ra, Dec: dec}, obs.Lat, obs.Lng, gst)
	return Position{
		Horizontal: astronomy.Horizontal{Altitude: hz.Altitude, Azimuth: hz.Azimuth},
		RA:         ra,
		Dec:        dec,
		RangeKm:    km,
	}, nil
}
