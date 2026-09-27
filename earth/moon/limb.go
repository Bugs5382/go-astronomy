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
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// BrightLimb is the orientation of the Moon's lit side for an observer: which
// way the bright limb faces, on the sky and on the observer's screen. All three
// angles are in degrees and turn counter-clockwise as the observer sees the
// sky, so 90 is to the left of the reference direction and 270 to the right.
type BrightLimb struct {
	// PositionAngle is the position angle of the midpoint of the bright limb,
	// in [0, 360), measured from the direction of the north celestial pole
	// through east (Meeus 48.5). It is a property of the sky and does not
	// depend on how the observer holds their head.
	PositionAngle float64
	// Parallactic is the parallactic angle at the Moon, in (-180, 180]: the
	// angle from the direction of the north celestial pole to the direction
	// of the zenith, measured the same way (Meeus 14.1). It is negative while
	// the Moon is east of the meridian and positive after it has crossed.
	Parallactic float64
	// ZenithAngle is PositionAngle - Parallactic, in [0, 360): the direction
	// of the bright limb measured from "up" on the observer's sky, the
	// direction toward the zenith. For a screen whose top is the observer's
	// up, 0 means the lit side faces the top of the screen, 90 the left, 180
	// the bottom, and 270 the right.
	ZenithAngle float64
}

// BrightLimbAt returns which way the Moon's bright limb faces for the observer
// at instant t (issue 46). It is the direction from the Moon toward the Sun on
// the observer's sky, from both bodies' topocentric apparent places, referred
// to the true equator and equinox of date; it points at the Sun even when the
// Sun is below the horizon.
//
// The angles are always defined, because the Sun and the Moon never coincide
// exactly, but near New and Full Moon they stop meaning anything a viewer
// could see: with less than about 0.1% or more than about 99.9% of the disc
// lit (see Illumination), there is no visible bright limb to orient, and the
// angle swings quickly as the Moon passes the Sun or the anti-Sun point. A
// caller drawing the Moon should treat it as undefined there.
//
// It returns a go-apperr coded error when the observer's latitude or
// longitude is out of range.
func BrightLimbAt(obs astronomy.Observer, t time.Time) (BrightLimb, error) {
	if err := validateObserver(obs); err != nil {
		return BrightLimb{}, err
	}
	ra, dec, _, gst := topocentricEquatorial(obs, t)

	// The Sun seen from the Moon lies on the great circle through the Moon and
	// the Sun seen by the observer (the Moon-to-Sun vector is in the plane of
	// the observer, the Moon, and the Sun), so the observer's topocentric Sun
	// gives the same position angle.
	sunRA, sunDec, sunKm := ephemeris.SunApparent(julian.TT(t))
	sunRA, sunDec, _ = ephemeris.Topocentric(sunRA, sunDec, sunKm, obs.Lat, 0, gst+obs.Lng)

	chi := coordinates.PositionAngle(sunRA, sunDec, ra, dec)
	q := parallacticAngle(obs.Lat, dec, gst+obs.Lng-ra)
	return BrightLimb{
		PositionAngle: chi,
		Parallactic:   q,
		ZenithAngle:   angles.Normalize(chi - q),
	}, nil
}

// parallacticAngle returns the parallactic angle, in degrees in (-180, 180],
// of a body at declination decDeg and local hour angle hourAngleDeg for an
// observer at latitude latDeg (Meeus 14.1). The form with the sine and cosine
// of the latitude, rather than its tangent, stays finite at the poles.
func parallacticAngle(latDeg, decDeg, hourAngleDeg float64) float64 {
	phi := angles.DegToRad(latDeg)
	d := angles.DegToRad(decDeg)
	h := angles.DegToRad(hourAngleDeg)
	q := math.Atan2(math.Sin(h)*math.Cos(phi), math.Sin(phi)*math.Cos(d)-math.Cos(phi)*math.Sin(d)*math.Cos(h))
	return angles.RadToDeg(q)
}
