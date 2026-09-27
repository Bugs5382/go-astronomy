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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/sun"
)

// SunPosition returns the Sun's horizontal position as seen from the Earth
// observer at instant t: the direction to the center of the disc paired with the
// disc's apparent angular diameter. The altitude is geometric and does not
// include atmospheric refraction, which the twilight segmentation folds into its
// horizon threshold instead. The position is computed on Terrestrial Time and
// accounts for nutation, aberration, and the observer's parallax on the
// flattened Earth.
//
// This is the Earth vantage on the Sun. Every part of it is Earth-specific: the
// Sun's apparent right ascension and declination come from the Earth's orbit
// (the solar model is Earth-based), the horizontal transform needs the
// observer's latitude, longitude, and Earth sidereal time, and the apparent
// diameter follows from the Earth-Sun distance. Only the size-versus-distance
// relation, sun.ApparentDiameter, is universal Sun physics.
func SunPosition(obs astronomy.Observer, t time.Time) astronomy.Position {
	return sunHorizontal(obs, t, sunPlaceAt(t))
}

// sunPlace is the Sun's apparent geocentric place at one instant: right
// ascension and declination of date in degrees, distance in km, and the
// nutation in right ascension (the equation of the equinoxes) in degrees,
// which turns mean sidereal time into apparent. It is everything about the
// Sun that does not depend on the observer.
type sunPlace struct {
	ra, dec, distKm, eqEquinoxes float64
}

// sunPlaceAt computes the Sun's place at t from VSOP87 and the nutation.
func sunPlaceAt(t time.Time) sunPlace {
	// The solar theory (VSOP87) runs on Terrestrial Time; the Earth's rotation
	// runs on UT, taken as UTC (issue 45).
	jde := julian.TT(t)
	// One nutation for the apparent longitude, the true obliquity, and the
	// apparent sidereal time.
	nut := ephemeris.NutationAt(jde)
	ra, dec, distKm := ephemeris.SunApparentNutated(jde, nut)
	return sunPlace{ra: ra, dec: dec, distKm: distKm, eqEquinoxes: nut.InRA()}
}

// sunHorizontal turns the Sun's place into the observer's horizon at t.
func sunHorizontal(obs astronomy.Observer, t time.Time, p sunPlace) astronomy.Position {
	gst := angles.Normalize(julian.GreenwichSiderealTime(t) + p.eqEquinoxes)

	// Move the geocentric place to the observer: the solar parallax is up to
	// 8.8 arc seconds. The disc stays sized from the Earth-Sun distance: the
	// observer's offset changes it by under 5e-5 of itself.
	ra, dec, _ := ephemeris.Topocentric(p.ra, p.dec, p.distKm, obs.Lat, heightMeters(obs), gst+obs.Lng)
	hz := coordinates.EquatorialToHorizontal(
		coordinates.Equatorial{RA: ra, Dec: dec},
		obs.Lat, obs.Lng, gst,
	)

	distanceAU := p.distKm / ephemeris.KmPerAU

	return astronomy.Position{
		Horizontal: astronomy.Horizontal{
			Altitude: hz.Altitude,
			Azimuth:  hz.Azimuth,
		},
		Diameter: sun.ApparentDiameter(distanceAU),
	}
}

// SunSample is one point on the Sun's arc as seen from Earth: the instant, the
// geometric altitude and azimuth in degrees, and TimeProgress, the fraction of
// the sampled span elapsed at this point, running from 0 at the first sample to
// 1 at the last.
type SunSample struct {
	Time         time.Time
	Altitude     float64
	Azimuth      float64
	TimeProgress float64
}

// SunTrack samples the Sun's arc across the civil day containing date, resolved
// in the observer's time zone from local midnight to the following local
// midnight. The day-of-month is taken from date; its time-of-day is ignored.
// Exactly samples points are returned, evenly spaced in time and inclusive of
// both endpoints, so the first sample sits at the start of the day and the last
// at its end. The span honors daylight-saving transitions, spanning 23 or 25
// hours on such days. Fewer than two samples cannot define a progress span, so
// SunTrack returns nil in that case.
//
// With more samples than the day has hours, the Sun's place is interpolated
// from its places at whole hours, which moves a sample by under 1e-9 degrees
// from SunPosition at the same instant and makes a fine track several times
// cheaper.
func SunTrack(obs astronomy.Observer, date time.Time, samples int) []SunSample {
	if samples < 2 {
		return nil
	}
	loc := obs.Location()
	local := date.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	span := end.Sub(start)

	position := SunPosition
	if samples > trackInterpolateAbove {
		places := newSunPlaces()
		position = func(obs astronomy.Observer, t time.Time) astronomy.Position {
			return sunHorizontal(obs, t, places.at(t))
		}
	}

	out := make([]SunSample, samples)
	last := samples - 1
	for i := range out {
		progress := float64(i) / float64(last)
		var when time.Time
		if i == last {
			when = end // exact endpoint, free of rounding drift
		} else {
			when = start.Add(time.Duration(progress * float64(span)))
		}
		pos := position(obs, when)
		out[i] = SunSample{
			Time:         when,
			Altitude:     pos.Altitude,
			Azimuth:      pos.Azimuth,
			TimeProgress: progress,
		}
	}
	return out
}

// trackInterpolateAbove is the sample count above which SunTrack interpolates
// the Sun's place: a day spans about 28 hourly places with the margins the
// interpolation needs, so below that direct positions are cheaper.
const trackInterpolateAbove = 32

// heightMeters is the observer's height for the parallax, in metres, with a
// NaN or infinite height (which the error-returning functions reject) taken
// as sea level by the functions that have no error to report.
func heightMeters(obs astronomy.Observer) float64 {
	if obs.Height.Err() != nil {
		return 0
	}
	return obs.Height.Meters()
}
