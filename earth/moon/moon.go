// Package moon provides the Earth vantage on Luna, the Earth's moon: its
// topocentric horizontal position and apparent size, its phase and illumination,
// its rise and set for an observer, and an arc track over a time span.
//
// The Moon belongs to Earth in this library's architecture, so the package lives
// under earth. Every quantity here is Earth-specific: the lunar coordinates come
// from the meeus Earth-based lunar theory, the horizontal transform needs the
// observer's latitude, longitude, and Earth sidereal time, the apparent diameter
// follows from the Earth-Moon distance, and the refraction model is Earth's.
//
// Positions are the center of the disc, in degrees, paired with the apparent
// angular diameter so a consumer can size and place the disc and reason about
// alignment and overlap with other bodies. Altitudes from Position are geometric
// (no refraction) and topocentric (corrected for the observer's displacement
// from the Earth's center by lunar parallax); ApparentPosition adds atmospheric
// refraction. The package is stateless and concurrency-safe: time is always a
// parameter, never captured. Accuracy is amateur, arcminute-class; nutation,
// delta-T, and Earth flattening are below that floor and are not modeled.
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
	"errors"
	"math"
	"time"

	"github.com/soniakeys/meeus/v3/moonposition"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// moonRadiusRatio is the ratio of the Moon's radius to the Earth's equatorial
// radius (Meeus, Astronomical Algorithms, chapter 55). The Moon's apparent
// angular semidiameter is this fraction of its equatorial horizontal parallax.
const moonRadiusRatio = 0.272481

// refractionFloor is the geometric altitude, in degrees, below which
// ApparentPosition stops applying atmospheric refraction. Below the horizon the
// Moon is not visible, and Bennett's refraction model, defined for apparent
// altitudes, loses meaning a few degrees under the horizon, so refraction is
// held at zero there. This keeps the apparent altitude at or above the geometric
// altitude everywhere.
const refractionFloor = -1.0

// Sentinel causes wrapped by the observer-taking functions. Each is returned
// inside a go-apperr coded error, so errors.Is keeps matching these values while
// apperr.Code recovers the stable numeric code from the astronomy package's code
// registry. Match a condition with errors.Is(err, ErrInvalid...) or branch on
// the code with apperr.Code(err).
var (
	// ErrInvalidLatitude is the cause when the observer's latitude is outside
	// the physical range [-90, 90]; the coded error carries
	// astronomy.CodeInvalidLatitude.
	ErrInvalidLatitude = errors.New("moon: observer latitude out of range")
	// ErrInvalidLongitude is the cause when the observer's longitude is outside
	// the range [-180, 180]; the coded error carries
	// astronomy.CodeInvalidLongitude.
	ErrInvalidLongitude = errors.New("moon: observer longitude out of range")
)

// validateObserver reports the coded error for an out-of-range observer, or nil
// when the latitude and longitude are both physical. It centralizes the input
// contract shared by every function that takes an observer.
func validateObserver(obs astronomy.Observer) error {
	if obs.Lat < -90 || obs.Lat > 90 {
		return apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if obs.Lng < -180 || obs.Lng > 180 {
		return apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	return nil
}

// topocentric returns the Moon's topocentric horizontal coordinates (geometric,
// without refraction) and its apparent angular semidiameter in degrees at t. The
// geocentric ecliptic position from the meeus lunar theory is rotated into the
// equatorial frame with the mean obliquity, transformed to the observer's
// horizontal frame, then lowered by the parallax in altitude to account for the
// observer's displacement from the Earth's center. The parallax in altitude is
// approximated as the equatorial horizontal parallax times the cosine of the
// altitude, adequate at the library's arcminute-class accuracy.
func topocentric(obs astronomy.Observer, t time.Time) (coordinates.Horizontal, float64) {
	jde := julian.Date(t)
	lam, bet, dist := moonposition.Position(jde)
	obl := julian.MeanObliquity(t)
	eq := coordinates.EclipticToEquatorial(
		coordinates.Ecliptic{Lon: lam.Deg(), Lat: bet.Deg()}, obl)
	gst := julian.GreenwichSiderealTime(t)
	hz := coordinates.EquatorialToHorizontal(eq, obs.Lat, obs.Lng, gst)

	parDeg := moonposition.Parallax(dist).Deg()
	hz.Altitude -= parDeg * math.Cos(angles.DegToRad(hz.Altitude))

	semiDeg := moonRadiusRatio * parDeg
	return hz, semiDeg
}

// position builds the public Position value from the topocentric coordinates.
func position(obs astronomy.Observer, t time.Time) astronomy.Position {
	hz, semiDeg := topocentric(obs, t)
	return astronomy.Position{
		Horizontal: astronomy.Horizontal{
			Altitude: hz.Altitude,
			Azimuth:  hz.Azimuth,
		},
		Diameter: astronomy.AngularDiameter(2 * semiDeg),
	}
}

// Position returns the Moon's topocentric horizontal position as seen from the
// observer at instant t: the direction to the center of the disc paired with the
// disc's apparent angular diameter. The altitude is geometric and does not
// include atmospheric refraction; use ApparentPosition for the refracted
// altitude. It returns a go-apperr coded error when the observer's latitude or
// longitude is out of range.
func Position(obs astronomy.Observer, t time.Time) (astronomy.Position, error) {
	if err := validateObserver(obs); err != nil {
		return astronomy.Position{}, err
	}
	return position(obs, t), nil
}

// ApparentPosition is Position with atmospheric refraction applied to the
// altitude, using the Earth refraction model. Refraction lifts the apparent
// altitude above the geometric altitude while the Moon is at or above the
// horizon and is not applied once the Moon is well below it, where it is not
// visible. The azimuth and diameter are unchanged from Position.
func ApparentPosition(obs astronomy.Observer, t time.Time) (astronomy.Position, error) {
	if err := validateObserver(obs); err != nil {
		return astronomy.Position{}, err
	}
	pos := position(obs, t)
	if pos.Altitude >= refractionFloor {
		if r := earth.Refraction(pos.Altitude); r > 0 {
			pos.Altitude += r
		}
	}
	return pos, nil
}
