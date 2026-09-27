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
	"sync/atomic"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/star"
)

// StarPosition returns a star's horizontal position, its altitude and azimuth in
// degrees, as seen from the Earth observer at instant t. The star is carried
// along its space motion from the catalog epoch (star.Star.PositionAt), then
// reduced to its apparent place on the true equator and equinox of date
// (precession, nutation, the annual parallax of a star with a distance, and the
// annual aberration of the Earth's motion) and rotated into the observer's
// horizon by the apparent sidereal time, so the position tracks the sky's
// diurnal rotation, its precessional drift, and the star's own motion.
//
// This is the Earth vantage on a star. The star's right ascension,
// declination, distance, and motion are universal (they live in the star
// package); everything applied here is Earth-specific: the Earth's precession,
// nutation, position, and orbital velocity, its sidereal time, and the
// observer's latitude and longitude. The altitude is geometric and does not
// include atmospheric refraction. Use the returned Horizontal's AboveHorizon
// method to test visibility.
//
// Against the IAU SOFA library the direction agrees to about 0.1 arc second
// around the present, drifting to about 0.35 by 1950 or 2100 with the IAU 1976
// precession. The diurnal aberration of the observer's rotation (up to 0.3 arc
// second) and the Sun's bending of starlight (under 0.01 arc second more than
// 45 degrees from the Sun) are not applied. It returns a go-apperr coded error
// when the observer's latitude or longitude is out of range.
//
// Most of the work is the instant, not the star. StarPosition remembers the
// last instant's frame, so a loop over many stars at one instant pays for it
// once; a StarField makes that explicit.
func StarPosition(s star.Star, obs astronomy.Observer, t time.Time) (astronomy.Horizontal, error) {
	f, err := NewStarField(obs, t)
	if err != nil {
		return astronomy.Horizontal{}, err
	}
	return f.Position(s), nil
}

// StarField is the Earth's view of the stars for one observer at one
// instant: the precession, nutation, and the Earth's position and velocity
// that every star's apparent place shares, and the apparent sidereal time.
// Building one costs a few microseconds; each star after that about as much
// as a v1.0 star position, so a whole sky shares one. It holds one instant
// and is safe for concurrent use; build another for another instant.
type StarField struct {
	obs   astronomy.Observer
	t     time.Time
	frame ephemeris.ApparentFrame
	gst   float64
}

// NewStarField prepares the stars for the observer at instant t. It returns
// a go-apperr coded error when the observer's latitude or longitude is out of
// range.
func NewStarField(obs astronomy.Observer, t time.Time) (*StarField, error) {
	if err := validateObserver(obs); err != nil {
		return nil, err
	}
	at := frameAt(t)
	return &StarField{obs: obs, t: t, frame: at.frame, gst: at.gst}, nil
}

// instantFrame is the part of a star's place that depends only on the
// instant.
type instantFrame struct {
	sec   int64
	nsec  int
	frame ephemeris.ApparentFrame
	gst   float64
}

// lastFrame remembers the most recent instant's frame. It is a memo, not
// state: the frame is a pure function of the instant, so a hit returns
// exactly what a fresh computation would, and concurrent callers at other
// instants only replace it.
var lastFrame atomic.Pointer[instantFrame]

// frameAt returns the frame at instant t, from lastFrame when it holds t.
func frameAt(t time.Time) *instantFrame {
	sec, nsec := t.Unix(), t.Nanosecond()
	if f := lastFrame.Load(); f != nil && f.sec == sec && f.nsec == nsec {
		return f
	}
	jde := julian.TT(t)
	dpsi, deps := ephemeris.Nutation(jde)
	f := &instantFrame{
		sec:   sec,
		nsec:  nsec,
		frame: ephemeris.NewApparentFrame(jde, dpsi, deps),
		gst:   julian.ApparentSiderealTime(t),
	}
	lastFrame.Store(f)
	return f
}

// Position returns the star's altitude and azimuth in degrees, as
// StarPosition does, for the field's observer and instant.
func (f *StarField) Position(s star.Star) astronomy.Horizontal {
	ra, dec := s.PositionAt(f.t)
	ra, dec = f.frame.Apply(ra, dec, s.Distance)
	hz := coordinates.EquatorialToHorizontal(coordinates.Equatorial{RA: ra, Dec: dec}, f.obs.Lat, f.obs.Lng, f.gst)
	return astronomy.Horizontal{Altitude: hz.Altitude, Azimuth: hz.Azimuth}
}
