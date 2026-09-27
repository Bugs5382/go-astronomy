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
	"math"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Site is a place on a body: the body, the planetodetic latitude (the angle
// between the local vertical of the body's reference ellipsoid and its
// equator, as JPL Horizons takes it), the east longitude in the body's IAU
// frame, both in degrees, and the height above the reference ellipsoid.
//
// Longitude is east-positive on every body, as on an astronomy.Observer; the
// IAU's west-positive planetographic longitude on Mars converts as
// 360 - west (Jezero, 282.55 W, is 77.45 E). A Site has no time zone: time
// zones are an Earth institution, and every instant is a time.Time, returned
// in UTC.
type Site struct {
	// Body is the body the site stands on. The Sun is not a valid site.
	Body *Body
	// Lat is the planetodetic latitude in degrees, positive north, in
	// [-90, 90].
	Lat float64
	// Lon is the east longitude in degrees, in [-180, 180].
	Lon float64
	// Height is the height above the body's reference ellipsoid, optional.
	// Off Earth it lowers the horizon by the geometric dip.
	Height astronomy.Height
}

// Result is a target as seen from a Site.
type Result struct {
	// Position is the direction to the centre of the target's disc in the
	// site's local horizon (geometric altitude, without refraction, and
	// azimuth clockwise from the body's north, in degrees), paired with the
	// apparent angular diameter. The direction is apparent: corrected for
	// light-time and for the aberration of the site's motion.
	astronomy.Position
	// DistanceAU is the distance from the site, in astronomical units.
	DistanceAU float64
	// LightTime is how long ago the light now arriving left the target.
	LightTime time.Duration
	// PhaseAngle is the Sun-target-site angle in degrees: 0 when the disc is
	// fully lit, 180 when the lit side faces away. It is 0 for the Sun.
	PhaseAngle float64
	// Illuminated is the lit fraction of the disc, (1 + cos PhaseAngle) / 2.
	Illuminated float64
	// Elongation is the angle between the target and the Sun as seen from
	// the site, in degrees, [0, 180]. It is 0 for the Sun.
	Elongation float64
}

// validate checks the site and the target, returning the coded error of the
// first problem.
func validate(site Site, target *Body) error {
	if site.Body == nil || target == nil || site.Body.kind == kindSun {
		return apperr.Coded(astronomy.CodeUnknownBody, ErrUnknownBody)
	}
	if site.Body.Name() == target.Name() {
		return apperr.Coded(astronomy.CodeSameBody, ErrSameBody)
	}
	return validateSite(site)
}

// validateSite checks the site alone.
func validateSite(site Site) error {
	if site.Body == nil || site.Body.kind == kindSun {
		return apperr.Coded(astronomy.CodeUnknownBody, ErrUnknownBody)
	}
	if !(site.Lat >= -90 && site.Lat <= 90) {
		return apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if !(site.Lon >= -180 && site.Lon <= 180) {
		return apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	return site.Height.Err()
}

// observer is the site as an astronomy.Observer, for a site on Earth.
func (s Site) observer() astronomy.Observer {
	return astronomy.Observer{Lat: s.Lat, Lng: s.Lon, TZ: time.UTC, Height: s.Height}
}

// Position returns the target as seen from the site at instant t: its place
// in the site's local horizon with its apparent diameter, its distance and
// light-time, its phase, and its elongation from the Sun.
//
// On Earth the altitude, azimuth, and diameter are exactly those of
// earth.SunPosition, moon.Position, and the planet packages' Position, so an
// Earth site gives the v1 answers. Off Earth the target's heliocentric
// position (VSOP87, and the Meeus Moon) is differenced with the site's and
// turned into the site's horizon through the body's IAU pole and prime
// meridian. The light-time and the aberration of the site's motion are
// applied; the gravitational bending of light is not (under 0.01 arc second
// away from the Sun's limb).
//
// It returns a coded error for a missing body (ErrUnknownBody), a target
// that is the site's own body (ErrSameBody), or an out-of-range site.
func Position(site Site, target *Body, t time.Time) (Result, error) {
	if err := validate(site, target); err != nil {
		return Result{}, err
	}
	l := site.look(target, julian.TT(t))
	res := l.result()
	if site.Body.kind != kindEarth {
		return res, nil
	}
	// On Earth the direction and the diameter come from the v1 packages.
	obs := site.observer()
	switch target.kind {
	case kindSun:
		res.Position = earth.SunPosition(obs, t)
	case kindMoon:
		pos, err := moon.Position(obs, t)
		if err != nil {
			return Result{}, err
		}
		res.Position = pos
	case kindPlanet:
		r, err := target.planet.Position(obs, t)
		if err != nil {
			return Result{}, err
		}
		res = Result{
			Position:    r.Position,
			DistanceAU:  r.DistanceAU,
			LightTime:   r.LightTime,
			PhaseAngle:  r.PhaseAngle,
			Illuminated: r.Illuminated,
			Elongation:  r.Elongation,
		}
	}
	return res, nil
}

// look is the geometry of one target from one site at one instant.
type look struct {
	alt, az   float64 // degrees
	semi      float64 // apparent semidiameter, degrees
	distAU    float64
	tauDays   float64
	phase     float64 // degrees
	elong     float64 // degrees
	hourAngle float64 // the target's local hour angle, degrees in (-180, 180]
	target    kind
}

func (l look) result() Result {
	return Result{
		Position: astronomy.Position{
			Horizontal: astronomy.Horizontal{Altitude: l.alt, Azimuth: l.az},
			Diameter:   astronomy.AngularDiameter(2 * l.semi),
		},
		DistanceAU:  l.distAU,
		LightTime:   time.Duration(l.tauDays * 86400 * float64(time.Second)),
		PhaseAngle:  l.phase,
		Illuminated: (1 + math.Cos(l.phase*radPerDeg)) / 2,
		Elongation:  l.elong,
	}
}

// velocityStep is the half-interval, in days, of the central difference that
// gives the site's velocity for the aberration.
const velocityStep = 1e-3

// siteHelio returns the site's heliocentric position on the J2000 equator at
// jde, and the rotation from that frame to the body's.
func (s Site) siteHelio(jde float64, bf vec) (vec, mat) {
	m := s.Body.rotation(jde)
	return s.Body.helio(jde).add(m.transpose().apply(bf).scale(1 / kmPerAU)), m
}

// look places the target for the site at the Julian ephemeris day jde.
func (s Site) look(target *Body, jde float64) look {
	bf, up, east, north := geodetic(s.Body.model, s.Lat, s.Lon, s.Height.Meters()/1000)
	obs, rot := s.siteHelio(jde, bf)
	ahead, _ := s.siteHelio(jde+velocityStep, bf)
	behind, _ := s.siteHelio(jde-velocityStep, bf)
	vel := ahead.sub(behind).scale(1 / (2 * velocityStep)) // AU per day

	// Light-time: the target where the light now arriving left it.
	tau := 0.0
	var tgt, rho vec
	for range 5 {
		tgt = target.helio(jde - tau)
		rho = tgt.sub(obs)
		next := rho.norm() / lightDayAU
		done := math.Abs(next-tau) < 1e-10
		tau = next
		if done {
			break
		}
	}
	dist := rho.norm()
	// Aberration of the site's motion, to first order in v/c.
	dir := rho.unit().add(vel.scale(1 / lightDayAU)).unit()

	d := rot.apply(dir)
	l := look{
		alt:     math.Asin(clamp(d.dot(up))) / radPerDeg,
		az:      math.Mod(math.Atan2(d.dot(east), d.dot(north))/radPerDeg+360, 360),
		semi:    math.Asin(clamp(target.radiusKm/(dist*kmPerAU))) / radPerDeg,
		distAU:  dist,
		tauDays: tau,
		target:  target.kind,
	}
	lonT := math.Atan2(d[1], d[0]) / radPerDeg
	l.hourAngle = math.Mod(s.Lon-lonT+540, 360) - 180
	if target.kind != kindSun {
		toSun := tgt.scale(-1)
		toSite := obs.sub(tgt)
		l.phase = angle(toSun, toSite)
		l.elong = angle(rho, obs.scale(-1))
	}
	return l
}

// angle returns the angle between two vectors in degrees.
func angle(a, b vec) float64 {
	return math.Acos(clamp(a.dot(b)/(a.norm()*b.norm()))) / radPerDeg
}

func clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
