package satellite

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

// Look angles: a satellite as an observer sees it at one instant.

import (
	"errors"
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Look is a satellite as an observer sees it at one instant.
type Look struct {
	// Horizontal is the geometric (airless) altitude and azimuth, in
	// degrees, azimuth clockwise from true north.
	astronomy.Horizontal
	// RangeKm is the distance from the observer.
	RangeKm float64
	// RangeRateKmS is the rate of change of the range, positive when the
	// satellite is moving away.
	RangeRateKmS float64
	// Sunlit reports that the centre of the Sun is visible from the
	// satellite, so it is outside the Earth's umbra and at least half lit
	// through any penumbra.
	Sunlit bool
	// SunAltitude is the Sun's geometric altitude for the observer, in
	// degrees; the observer is in darkness when it is well below zero.
	SunAltitude float64
	// PhaseAngle is the Sun-satellite-observer angle, in degrees: 0 when the
	// observer sees the fully lit side.
	PhaseAngle float64
	// Latitude, Longitude, and AltitudeKm are the geodetic sub-satellite point
	// and height above the WGS84 ellipsoid.
	Latitude, Longitude, AltitudeKm float64
}

// Position returns where the satellite is in the observer's sky at t. The
// element set's age is the caller's to judge (Elements.Age): SGP4 positions
// for a low orbit degrade by kilometres a day after epoch. Errors are
// go-apperr coded: an out-of-range observer, or astronomy.CodeSatellitePropagation
// when SGP4 fails (for a decayed satellite the Look is still filled in).
func Position(obs astronomy.Observer, e Elements, t time.Time) (Look, error) {
	if err := validateObserver(obs); err != nil {
		return Look{}, err
	}
	return look(obs, e, t)
}

// look is Position without the observer check.
func look(obs astronomy.Observer, e Elements, t time.Time) (Look, error) {
	r, v, err := e.Propagate(t)
	if err != nil && !errors.Is(err, ErrDecayed) {
		return Look{}, err
	}

	// TEME to the Earth-fixed frame by the rotation through GMST 1982, the
	// angle SGP4 defines TEME against. Polar motion (metres) is ignored, and
	// UT1 is taken as UTC.
	g := gmst82(julian.Date(t))
	cg, sg := math.Cos(g), math.Sin(g)
	rf := [3]float64{cg*r[0] + sg*r[1], -sg*r[0] + cg*r[1], r[2]}
	vf := [3]float64{
		cg*v[0] + sg*v[1] + earthRotationRadS*rf[1],
		-sg*v[0] + cg*v[1] - earthRotationRadS*rf[0],
		v[2],
	}

	site := siteECEF(obs)
	rho := [3]float64{rf[0] - site[0], rf[1] - site[1], rf[2] - site[2]}
	rng := norm(rho)
	lat, lng := obs.Lat*math.Pi/180, obs.Lng*math.Pi/180
	sl, cl := math.Sin(lat), math.Cos(lat)
	so, co := math.Sin(lng), math.Cos(lng)
	south := sl*co*rho[0] + sl*so*rho[1] - cl*rho[2]
	east := -so*rho[0] + co*rho[1]
	up := cl*co*rho[0] + cl*so*rho[1] + sl*rho[2]

	alt := math.Asin(up/rng) * 180 / math.Pi
	az := math.Atan2(east, -south) * 180 / math.Pi
	if az < 0 {
		az += 360
	}

	// The Sun, in TEME to the precision the shadow needs: the apparent place
	// of date, whose equinox differs from TEME's by the equation of the
	// equinoxes (about a second of right ascension).
	sunRA, sunDec, sunKm := ephemeris.SunApparent(julian.TT(t))
	sun := fromRADec(sunRA, sunDec, sunKm)
	siteTEME := [3]float64{cg*site[0] - sg*site[1], sg*site[0] + cg*site[1], site[2]}

	toSun := sub(sun, r)
	toObs := sub(siteTEME, r)
	phase := angleBetween(toSun, toObs) * 180 / math.Pi

	sgLat, sgLng, sgAlt := geodetic(rf)

	out := Look{
		Horizontal:   astronomy.Horizontal{Altitude: alt, Azimuth: az},
		RangeKm:      rng,
		RangeRateKmS: dot(rho, vf) / rng,
		Sunlit:       sunlit(r, sun),
		SunAltitude:  earth.SunPosition(obs, t).Altitude,
		PhaseAngle:   phase,
		Latitude:     sgLat,
		Longitude:    sgLng,
		AltitudeKm:   sgAlt,
	}
	return out, err
}

// sunlit reports whether the centre of the Sun is above the Earth's limb as
// seen from the satellite at r (the Earth taken as a sphere of equatorial
// radius), with the Sun at sun, both in km in one frame.
func sunlit(r, sun [3]float64) bool {
	toEarth := [3]float64{-r[0], -r[1], -r[2]}
	toSun := sub(sun, r)
	earthRadius := math.Asin(math.Min(1, wgs84A/norm(r)))
	return angleBetween(toEarth, toSun) > earthRadius
}
