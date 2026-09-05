// Package sun provides universal solar geometry: the Sun's geometric horizontal
// position and its arc across a day, for any observer and instant. It is
// deliberately body-neutral and Earth-agnostic. Altitudes are geometric, with
// no refraction applied, and no Earth-specific naming (civil or nautical
// twilight, sunrise, sunset) appears here; those conventions belong to a
// body-specific layer built on top of this package.
//
// All angles are in degrees. Azimuth is measured clockwise from true north and
// altitude is the angle above the geometric horizon. Positions describe the
// center of the Sun's disc and are paired with the disc's apparent angular
// diameter so a consumer can size, place, and reason about overlap between
// bodies. The package is stateless and concurrency-safe: time is always a
// parameter, never captured.
package sun

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
	"time"

	meeusbase "github.com/soniakeys/meeus/v3/base"
	"github.com/soniakeys/meeus/v3/solar"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// sunSemidiameterArcsecAt1AU is the Sun's apparent angular semidiameter, in arc
// seconds, at the unit Earth-Sun distance of one astronomical unit (Meeus,
// Astronomical Algorithms, chapter 55). The apparent semidiameter at any
// instant scales inversely with the Earth-Sun distance in AU.
const sunSemidiameterArcsecAt1AU = 959.63

// Position returns the Sun's geometric horizontal position for the observer at
// instant t: the direction to the center of the disc paired with the disc's
// apparent angular diameter. The altitude is geometric and does not include
// atmospheric refraction, which is a body or atmosphere concern applied by a
// higher layer. The apparent position accounts for nutation and aberration.
func Position(obs astronomy.Observer, t time.Time) astronomy.Position {
	jd := julian.Date(t)
	ra, dec := solar.ApparentEquatorial(jd)
	gst := julian.GreenwichSiderealTime(t)

	hz := coordinates.EquatorialToHorizontal(
		coordinates.Equatorial{RA: ra.Deg(), Dec: dec.Deg()},
		obs.Lat, obs.Lng, gst,
	)

	return astronomy.Position{
		Horizontal: astronomy.Horizontal{
			Altitude: hz.Altitude,
			Azimuth:  hz.Azimuth,
		},
		Diameter: angularDiameter(jd),
	}
}

// angularDiameter returns the Sun's apparent angular diameter, in degrees, for
// the given Julian Date, derived from the Earth-Sun distance. It is a property
// of the Sun and the instant alone, independent of the observer's location.
func angularDiameter(jd float64) astronomy.AngularDiameter {
	distanceAU := solar.Radius(meeusbase.J2000Century(jd))
	diameterArcsec := 2 * sunSemidiameterArcsecAt1AU / distanceAU
	return astronomy.AngularDiameter(diameterArcsec / 3600)
}

// Sample is one point on the Sun's arc: the instant, the geometric altitude and
// azimuth in degrees, and TimeProgress, the fraction of the sampled span
// elapsed at this point, running from 0 at the first sample to 1 at the last.
type Sample struct {
	Time         time.Time
	Altitude     float64
	Azimuth      float64
	TimeProgress float64
}

// Track samples the Sun's arc across the civil day containing date, resolved in
// the observer's time zone from local midnight to the following local midnight.
// The day-of-month is taken from date; its time-of-day is ignored. Exactly
// samples points are returned, evenly spaced in time and inclusive of both
// endpoints, so the first sample sits at the start of the day and the last at
// its end. The span honors daylight-saving transitions, spanning 23 or 25 hours
// on such days. Fewer than two samples cannot define a progress span, so Track
// returns nil in that case.
func Track(obs astronomy.Observer, date time.Time, samples int) []Sample {
	if samples < 2 {
		return nil
	}
	loc := obs.Location()
	local := date.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	span := end.Sub(start)

	out := make([]Sample, samples)
	last := samples - 1
	for i := range out {
		progress := float64(i) / float64(last)
		var when time.Time
		if i == last {
			when = end // exact endpoint, free of rounding drift
		} else {
			when = start.Add(time.Duration(progress * float64(span)))
		}
		pos := Position(obs, when)
		out[i] = Sample{
			Time:         when,
			Altitude:     pos.Altitude,
			Azimuth:      pos.Azimuth,
			TimeProgress: progress,
		}
	}
	return out
}
