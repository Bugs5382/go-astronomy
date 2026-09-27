package sky

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

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/iau"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/planet"
)

// Body is a body a Site can stand on and a target Position can place: the
// Sun, a planet, or the Moon. It carries what the sky from that body depends
// on: the IAU pole and prime meridian, the reference ellipsoid, the
// atmosphere's refraction, and whether that atmosphere makes a twilight. Use
// the package values Sun, Earth, and Moon, or Planet for a planet package's
// Planet. A Body is immutable and safe to share.
type Body struct {
	model *iau.Model
	// orient returns the rotation from the J2000 equator to the body's frame
	// at the Julian ephemeris day; nil means the IAU model.
	orient func(jde float64) mat
	// helio returns the body's heliocentric position at the Julian ephemeris
	// day, on the J2000 equator, in astronomical units.
	helio func(jde float64) vec
	// radiusKm sizes the body's disc as a target.
	radiusKm float64
	// refraction is the horizontal refraction in degrees; zero for a body
	// with no atmosphere, or one whose refraction is not modelled.
	refraction float64
	// twilight is the body's default day segmentation, or nil when its
	// atmosphere makes no twilight to divide.
	twilight *earth.Segmentation
	// planet is the planet package's Body, for a planet.
	planet planet.Body
	kind   kind
}

type kind int

const (
	kindPlanet kind = iota
	kindSun
	kindEarth
	kindMoon
)

// Sun is the Sun. It can be a target from any site; a Site cannot stand on
// it.
var Sun = &Body{
	model:    iau.Sun,
	helio:    func(float64) vec { return vec{} },
	radiusKm: iau.Sun.EquatorialKm,
	kind:     kindSun,
}

// Earth is the Earth. A Site on Earth resolves through the earth, earth/moon,
// and planet packages, so it gives exactly their answers; its twilight is
// earth.DefaultSegmentation and its horizontal refraction the standard 34 arc
// minutes.
var Earth = &Body{
	model:      iau.Earth,
	orient:     earthOrientation,
	helio:      earthHelio,
	radiusKm:   iau.Earth.EquatorialKm,
	refraction: 34.0 / 60,
	twilight:   &earth.DefaultSegmentation,
	kind:       kindEarth,
}

// Moon is Luna. It has no atmosphere: no refraction and no twilight, so a
// body sets when it geometrically sets and the lunar day divides only at the
// Sun's horizon crossings.
var Moon = &Body{
	model:    iau.Moon,
	helio:    moonHelio,
	radiusKm: iau.Moon.EquatorialKm,
	kind:     kindMoon,
}

// ErrUnknownBody is the cause when a Site or a target has no Body, or a
// planet has no IAU rotation model here; the coded error carries
// astronomy.CodeUnknownBody.
var ErrUnknownBody = errors.New("sky: unknown or missing body")

// ErrSameBody is the cause when the target is the body the site stands on;
// the coded error carries astronomy.CodeSameBody.
var ErrSameBody = errors.New("sky: target is the body the site stands on")

// ErrInvalidLatitude and ErrInvalidLongitude are the causes when a site's
// latitude is outside [-90, 90] or its longitude outside [-180, 180]; the
// coded errors carry astronomy.CodeInvalidLatitude and
// astronomy.CodeInvalidLongitude.
var (
	ErrInvalidLatitude  = errors.New("sky: site latitude out of range")
	ErrInvalidLongitude = errors.New("sky: site longitude out of range")
)

// Planet returns the Body of a planet package's Planet, for example
// Planet(mars.Planet). Its position comes from that package's own VSOP87
// table, so only the planets a program imports are linked. It returns a coded
// ErrUnknownBody for a nil planet or one with no IAU rotation model.
//
// Planets carry no refraction and no twilight here. Mars has a thin
// atmosphere that refracts a body on the horizon by well under an arc minute
// and a long dusty twilight with no conventional levels; neither is modelled,
// and JPL Horizons treats Mars as airless too. The giant planets have no
// surface: a Site on one stands on the reference ellipsoid (the 1 bar level).
func Planet(p planet.Body) (*Body, error) {
	if p == nil {
		return nil, apperr.Coded(astronomy.CodeUnknownBody, ErrUnknownBody)
	}
	m := iau.ByName(p.Name())
	if m == nil || m == iau.Earth || m == iau.Moon || m == iau.Sun {
		return nil, apperr.Coded(astronomy.CodeUnknownBody, ErrUnknownBody)
	}
	return &Body{
		model: m,
		helio: func(jde float64) vec {
			h := p.Heliocentric(julian.FromTT(jde))
			return ofDateToJ2000(jde).apply(spherical(h.Lon, h.Lat, h.DistanceAU))
		},
		radiusKm: p.RadiusKm(),
		planet:   p,
		kind:     kindPlanet,
	}, nil
}

// rotation returns the rotation from the J2000 equator to the body's frame at
// jde.
func (b *Body) rotation(jde float64) mat {
	if b.orient != nil {
		return b.orient(jde)
	}
	return bodyFixed(b.model.At(jde))
}

// earthOrientation is the Earth's frame from the precession, the nutation,
// and the apparent sidereal time the earth package uses, not the IAU model,
// which leaves out the nutation and is good only to a few arc minutes.
func earthOrientation(jde float64) mat {
	t := (jde - j2000) / daysPerCentu
	zeta := (2306.2181*t + 0.30188*t*t + 0.017998*t*t*t) * arcsecToRad
	z := (2306.2181*t + 1.09468*t*t + 0.018203*t*t*t) * arcsecToRad
	theta := (2004.3109*t - 0.42665*t*t - 0.041833*t*t*t) * arcsecToRad
	precess := rot3(-z).mul(rot2(theta)).mul(rot3(-zeta))
	dpsi, deps := ephemeris.Nutation(jde)
	eps := ephemeris.MeanObliquity(jde)
	nutate := rot1(-(eps + deps) * radPerDeg).mul(rot3(-dpsi * radPerDeg)).mul(rot1(eps * radPerDeg))
	gast := julian.ApparentSiderealTime(julian.FromTT(jde))
	return rot3(gast * radPerDeg).mul(nutate).mul(precess)
}

// earthHelio is the Earth's heliocentric position from VSOP87.
func earthHelio(jde float64) vec {
	l, b, r := ephemeris.EarthHeliocentric(jde)
	return ofDateToJ2000(jde).apply(spherical(l, b, r))
}

// moonHelio is the Moon's heliocentric position: the Earth's plus the
// geocentric Moon of Meeus chapter 47, both on the ecliptic of date.
func moonHelio(jde float64) vec {
	l, b, r := ephemeris.EarthHeliocentric(jde)
	ml, mb, mkm := ephemeris.MoonPosition(jde)
	return ofDateToJ2000(jde).apply(spherical(l, b, r).add(spherical(ml, mb, mkm/kmPerAU)))
}

// Name returns the body's lowercase name, such as "mars".
func (b *Body) Name() string { return b.model.Name }

// RadiusKm returns the equatorial radius that sizes the body's disc.
func (b *Body) RadiusKm() float64 { return b.radiusKm }

// PolarRadiusKm returns the polar radius of the body's reference ellipsoid.
func (b *Body) PolarRadiusKm() float64 { return b.model.PolarKm }

// HorizonRefraction returns the refraction, in degrees, that lifts a body on
// this body's horizon: 34 arc minutes on Earth, and zero on an airless body
// or one whose refraction is not modelled.
func (b *Body) HorizonRefraction() float64 { return b.refraction }

// Twilight returns the body's default day segmentation and true, or false
// when its atmosphere makes no twilight: the day of such a body divides at
// the Sun's horizon crossings and nowhere else. Only the Earth has one.
func (b *Body) Twilight() (earth.Segmentation, bool) {
	if b.twilight == nil {
		return earth.Segmentation{}, false
	}
	return *b.twilight, true
}

// RotationPeriod returns the sidereal rotation period, negative for a
// retrograde spin (Venus, Uranus).
func (b *Body) RotationPeriod() time.Duration {
	return time.Duration(b.model.RotationDays * 86400 * float64(time.Second))
}

// SolarDay returns the mean solar day: the time between one noon and the
// next for a site on the body. It is 24 hours on Earth, a sol of 24h 39m 35s
// on Mars, a synodic month of 29.53 days on the Moon, and zero for the Sun.
func (b *Body) SolarDay() time.Duration {
	if b.kind == kindSun {
		return 0
	}
	return time.Duration(b.model.SolarDays() * 86400 * float64(time.Second))
}

// searchStep and searchWindow scale the rise and set search to the body: a
// step short enough that no crossing is skipped, and a window covering two
// full solar days and two rotations.
func (b *Body) searchStep() time.Duration {
	days := math.Min(math.Abs(b.model.RotationDays), b.model.SolarDays())
	return time.Duration(days / 96 * 86400 * float64(time.Second))
}

func (b *Body) searchWindow() time.Duration {
	days := 2.2 * math.Max(math.Abs(b.model.RotationDays), b.model.SolarDays())
	return time.Duration(days * 86400 * float64(time.Second))
}
