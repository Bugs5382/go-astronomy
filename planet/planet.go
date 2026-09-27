// Package planet places the planets from Mercury to Neptune in an observer's
// sky: topocentric apparent position, apparent diameter, phase, magnitude,
// elongation from the Sun, and rise, set, and transit. It also publishes each
// planet's heliocentric position, which does not depend on any observer.
//
// Positions come from the VSOP87 theory (Bretagnon and Francou 1988), version
// D, generated from the IMCCE files and truncated by amplitude; see the headers
// of the vsop87_*.go files. The reduction to an apparent place follows Meeus,
// Astronomical Algorithms, chapter 33: light-time and aberration by evaluating
// both the planet and the Earth at the instant the light left the planet, the
// FK5 correction (32.3), nutation, and the true obliquity, then the rigorous
// topocentric correction of chapter 40. The tables live in this package, so a
// program that never imports it never links them.
//
// Against JPL Horizons DE441 the topocentric positions agree to within about
// 1 arc second (see the package tests for the per-body figures). Magnitudes
// follow Mallama and Hilton (2018), the formulas the Astronomical Almanac and
// Horizons use, and agree with Horizons to within 0.1 magnitude.
//
// The package is stateless and concurrency-safe: time is always a parameter.
package planet

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
	"fmt"
	"math"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

//go:generate go run ../internal/cmd/genvsop -body mercury -var mercuryVSOP87D -pkg planet -out vsop87_mercury.go
//go:generate go run ../internal/cmd/genvsop -body venus -var venusVSOP87D -pkg planet -out vsop87_venus.go
//go:generate go run ../internal/cmd/genvsop -body mars -var marsVSOP87D -pkg planet -out vsop87_mars.go
//go:generate go run ../internal/cmd/genvsop -body jupiter -var jupiterVSOP87D -pkg planet -out vsop87_jupiter.go
//go:generate go run ../internal/cmd/genvsop -body saturn -var saturnVSOP87D -pkg planet -out vsop87_saturn.go
//go:generate go run ../internal/cmd/genvsop -body uranus -var uranusVSOP87D -pkg planet -out vsop87_uranus.go
//go:generate go run ../internal/cmd/genvsop -body neptune -var neptuneVSOP87D -pkg planet -out vsop87_neptune.go

// Body is a planet. Earth is included so its heliocentric position can be
// differenced with another planet's (the view of Earth from Mars, say), but it
// is not a valid target for Position or the rise and set functions.
type Body int

// The planets, in order from the Sun.
const (
	Mercury Body = iota + 1
	Venus
	Earth
	Mars
	Jupiter
	Saturn
	Uranus
	Neptune
)

// Bodies lists every planet other than Earth, in order from the Sun.
var Bodies = []Body{Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune}

// bodyData holds what the package needs about one planet.
type bodyData struct {
	name   string
	series *ephemeris.VSOP87Body
	// radiusKm is the IAU mean equatorial radius (Archinal et al. 2018), which
	// sizes the disc.
	radiusKm float64
	// nearSunDeg is the elongation below which the planet is flagged NearSun.
	nearSunDeg float64
}

var bodies = map[Body]bodyData{
	Mercury: {"mercury", &mercuryVSOP87D, 2440.53, 10},
	Venus:   {"venus", &venusVSOP87D, 6051.8, 5},
	Earth:   {"earth", nil, 6378.1366, 0},
	Mars:    {"mars", &marsVSOP87D, 3396.19, 11.5},
	Jupiter: {"jupiter", &jupiterVSOP87D, 71492, 9},
	Saturn:  {"saturn", &saturnVSOP87D, 60268, 11},
	Uranus:  {"uranus", &uranusVSOP87D, 25559, 15},
	Neptune: {"neptune", &neptuneVSOP87D, 24764, 15},
}

// String returns the planet's lowercase name, or "unknown".
func (b Body) String() string {
	if d, ok := bodies[b]; ok {
		return d.name
	}
	return "unknown"
}

// NearSunElongation returns the elongation from the Sun, in degrees, below
// which Position flags the planet NearSun. The values are the classical arcus
// visionis for the naked-eye planets (Mercury 10, Venus 5, Mars 11.5, Jupiter 9,
// Saturn 11), a rough guide to how far from the Sun a planet must be to be
// seen in twilight, and 15 degrees for Uranus and Neptune. They are heuristics:
// whether a planet is visible also depends on its magnitude, the sky, and the
// observer, which is why Position reports the elongation and magnitude as well.
func NearSunElongation(b Body) float64 {
	return bodies[b].nearSunDeg
}

// Sentinel causes, each returned inside a go-apperr coded error.
var (
	// ErrInvalidLatitude is the cause when the observer's latitude is outside
	// [-90, 90]; the coded error carries astronomy.CodeInvalidLatitude.
	ErrInvalidLatitude = errors.New("planet: observer latitude out of range")
	// ErrInvalidLongitude is the cause when the observer's longitude is
	// outside [-180, 180]; the coded error carries astronomy.CodeInvalidLongitude.
	ErrInvalidLongitude = errors.New("planet: observer longitude out of range")
	// ErrInvalidBody is the cause when a Body is not one of the planets, or is
	// Earth where a target is needed; the coded error carries
	// astronomy.CodeInvalidBody.
	ErrInvalidBody = errors.New("planet: not a planet that can be observed")
)

func validateObserver(obs astronomy.Observer) error {
	if obs.Lat < -90 || obs.Lat > 90 {
		return apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if obs.Lng < -180 || obs.Lng > 180 {
		return apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	return nil
}

func validateTarget(b Body) error {
	if d, ok := bodies[b]; !ok || d.series == nil {
		return apperr.Coded(astronomy.CodeInvalidBody, fmt.Errorf("%w: %d", ErrInvalidBody, int(b)))
	}
	return nil
}

// HeliocentricPosition is a planet's place as seen from the centre of the Sun,
// referred to the ecliptic and equinox of date (the dynamical frame of
// VSOP87D). It is geometric: no light-time or aberration, because it belongs
// to no observer.
type HeliocentricPosition struct {
	// Lon is the ecliptic longitude in degrees, [0, 360).
	Lon float64
	// Lat is the ecliptic latitude in degrees.
	Lat float64
	// DistanceAU is the distance from the Sun in astronomical units.
	DistanceAU float64
}

// Vector returns the position as rectangular ecliptic coordinates in
// astronomical units: x toward the equinox of date, z toward the north
// ecliptic pole. The view of one planet from another is the difference of
// their vectors at the same instant (before light-time).
func (h HeliocentricPosition) Vector() [3]float64 {
	l, b := angles.DegToRad(h.Lon), angles.DegToRad(h.Lat)
	return [3]float64{
		h.DistanceAU * math.Cos(b) * math.Cos(l),
		h.DistanceAU * math.Cos(b) * math.Sin(l),
		h.DistanceAU * math.Sin(b),
	}
}

// Heliocentric returns the planet's heliocentric position at instant t,
// including Earth's. It returns a coded ErrInvalidBody for a value that is not
// a planet.
func Heliocentric(b Body, t time.Time) (HeliocentricPosition, error) {
	d, ok := bodies[b]
	if !ok {
		return HeliocentricPosition{}, apperr.Coded(astronomy.CodeInvalidBody, fmt.Errorf("%w: %d", ErrInvalidBody, int(b)))
	}
	jde := julian.TT(t)
	var l, lat, r float64
	if d.series == nil {
		l, lat, r = ephemeris.EarthHeliocentric(jde)
	} else {
		l, lat, r = d.series.Heliocentric(jde)
	}
	return HeliocentricPosition{Lon: l, Lat: lat, DistanceAU: r}, nil
}

// Result is a planet as an observer sees it.
type Result struct {
	// Position is the topocentric direction to the centre of the disc
	// (geometric altitude, without refraction, and azimuth, in degrees),
	// paired with the apparent angular diameter, sized from the same
	// observer-to-planet distance.
	astronomy.Position
	// RA and Dec are the topocentric apparent right ascension, in [0, 360),
	// and declination, in degrees, referred to the true equator and equinox
	// of date.
	RA, Dec float64
	// DistanceAU is the distance from the observer, in astronomical units.
	DistanceAU float64
	// LightTime is how long ago the light now arriving left the planet.
	LightTime time.Duration
	// Magnitude is the apparent visual magnitude (Mallama and Hilton 2018).
	Magnitude float64
	// PhaseAngle is the Sun-planet-observer angle, in degrees: 0 when the disc
	// is fully lit.
	PhaseAngle float64
	// Illuminated is the fraction of the disc that is lit, in [0, 1].
	Illuminated float64
	// Elongation is the angle between the planet and the Sun as seen from the
	// Earth, in degrees, [0, 180].
	Elongation float64
	// NearSun reports that the elongation is below NearSunElongation for this
	// planet, so the planet is probably lost in the Sun's glare whatever its
	// altitude.
	NearSun bool
}

// Position returns the planet as seen by the observer at instant t. A negative
// altitude is below the horizon and is a valid answer. It returns a go-apperr
// coded error for an out-of-range observer or a Body that is not an observable
// planet.
func Position(obs astronomy.Observer, b Body, t time.Time) (Result, error) {
	if err := validateObserver(obs); err != nil {
		return Result{}, err
	}
	if err := validateTarget(b); err != nil {
		return Result{}, err
	}
	return position(obs, b, t), nil
}

// lightTimeDaysPerAU is the light-time for one astronomical unit, in days
// (Meeus 33.3).
const lightTimeDaysPerAU = 0.0057755183

// geocentric is the reduction of chapter 33 at one instant: the planet's
// apparent geocentric ecliptic place and the distances the phase and
// magnitude need.
type geocentric struct {
	lon, lat float64 // apparent, true equinox of date, degrees
	delta    float64 // Earth to planet, AU
	r        float64 // Sun to planet, AU
	earthR   float64 // Sun to Earth, AU
	tau      float64 // light-time, days
	lonGeom  float64 // before nutation and aberration are folded in, for Saturn's rings
	latGeom  float64
}

// geocentricAt reduces the planet to its apparent geocentric place at jde.
// Evaluating the Earth, as well as the planet, at the instant the light left
// the planet folds the aberration in with the light-time (Meeus, chapter 33).
func geocentricAt(d bodyData, jde float64) geocentric {
	_, _, earthR := ephemeris.EarthHeliocentric(jde)
	var g geocentric
	tau := 0.0
	var x, y, z, r float64
	for range 5 {
		l, b, rr := d.series.Heliocentric(jde - tau)
		l0, b0, r0 := ephemeris.EarthHeliocentric(jde - tau)
		lr, br := angles.DegToRad(l), angles.DegToRad(b)
		l0r, b0r := angles.DegToRad(l0), angles.DegToRad(b0)
		x = rr*math.Cos(br)*math.Cos(lr) - r0*math.Cos(b0r)*math.Cos(l0r)
		y = rr*math.Cos(br)*math.Sin(lr) - r0*math.Cos(b0r)*math.Sin(l0r)
		z = rr*math.Sin(br) - r0*math.Sin(b0r)
		r = rr
		next := lightTimeDaysPerAU * math.Sqrt(x*x+y*y+z*z)
		if math.Abs(next-tau) < 1e-9 {
			tau = next
			break
		}
		tau = next
	}
	g.tau = tau
	g.r = r
	g.earthR = earthR
	// The distance, and the phase and elongation that come from it, are from
	// the Earth at jde to the planet where the light left it. The Earth at
	// jde - tau above only carries the aberration into the direction.
	{
		l, b, rr := d.series.Heliocentric(jde - tau)
		l0, b0, r0 := ephemeris.EarthHeliocentric(jde)
		lr, br := angles.DegToRad(l), angles.DegToRad(b)
		l0r, b0r := angles.DegToRad(l0), angles.DegToRad(b0)
		dx := rr*math.Cos(br)*math.Cos(lr) - r0*math.Cos(b0r)*math.Cos(l0r)
		dy := rr*math.Cos(br)*math.Sin(lr) - r0*math.Cos(b0r)*math.Sin(l0r)
		dz := rr*math.Sin(br) - r0*math.Sin(b0r)
		g.delta = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	// The light-time is |planet(jde - tau) - Earth(jde - tau)| / c here, which
	// differs from the exact |planet(jde - tau) - Earth(jde)| / c by v/c of
	// itself, a second or two for Neptune: nothing a planet moves in.
	lon := angles.Normalize(angles.RadToDeg(math.Atan2(y, x)))
	lat := angles.RadToDeg(math.Atan2(z, math.Hypot(x, y)))
	g.lonGeom, g.latGeom = lon, lat
	dl, db := ephemeris.FK5Correction(lon, lat, jde)
	dpsi, _ := ephemeris.Nutation(jde)
	g.lon = angles.Normalize(lon + dl + dpsi)
	g.lat = lat + db
	return g
}

func position(obs astronomy.Observer, b Body, t time.Time) Result {
	d := bodies[b]
	jde := julian.TT(t)
	g := geocentricAt(d, jde)

	ra, dec := ephemeris.EclToEq(g.lon, g.lat, ephemeris.TrueObliquity(jde))
	gst := julian.ApparentSiderealTime(t)
	ra, dec, topoKm := ephemeris.Topocentric(ra, dec, g.delta*ephemeris.KmPerAU, obs.Lat, 0, gst+obs.Lng)
	hz := coordinates.EquatorialToHorizontal(coordinates.Equatorial{RA: ra, Dec: dec}, obs.Lat, obs.Lng, gst)

	// Phase angle and elongation from the triangle Sun, Earth, planet (Meeus
	// 41.2 and 48.2 in their distance form).
	cosI := (g.r*g.r + g.delta*g.delta - g.earthR*g.earthR) / (2 * g.r * g.delta)
	phase := angles.RadToDeg(math.Acos(clamp(cosI)))
	cosE := (g.earthR*g.earthR + g.delta*g.delta - g.r*g.r) / (2 * g.earthR * g.delta)
	elong := angles.RadToDeg(math.Acos(clamp(cosE)))

	diam := 2 * angles.RadToDeg(math.Asin(d.radiusKm/topoKm))

	return Result{
		Position: astronomy.Position{
			Horizontal: astronomy.Horizontal{Altitude: hz.Altitude, Azimuth: hz.Azimuth},
			Diameter:   astronomy.AngularDiameter(diam),
		},
		RA:          ra,
		Dec:         dec,
		DistanceAU:  topoKm / ephemeris.KmPerAU,
		LightTime:   time.Duration(g.tau * 86400 * float64(time.Second)),
		Magnitude:   magnitude(b, g, phase, jde),
		PhaseAngle:  phase,
		Illuminated: (1 + math.Cos(angles.DegToRad(phase))) / 2,
		Elongation:  elong,
		NearSun:     elong < d.nearSunDeg,
	}
}

func clamp(x float64) float64 { return math.Max(-1, math.Min(1, x)) }
