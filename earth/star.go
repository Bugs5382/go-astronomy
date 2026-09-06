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
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/star"
)

// starEpoch is the equinox of the star catalog's equatorial coordinates: J2000.0
// as a Julian year, the value the precession model expects.
const starEpoch = 2000.0

// StarPosition returns a star's horizontal position, its altitude and azimuth in
// degrees, as seen from the Earth observer at instant t. The star's J2000
// equatorial coordinates are precessed to the equinox of date and then rotated
// into the observer's horizontal frame using the Earth's sidereal time, so the
// position tracks the sky's diurnal rotation and its slow precessional drift.
//
// This is the Earth vantage on a star. The star's right ascension, declination,
// and distance are universal (they live in the star package); everything applied
// here is Earth-specific: the horizontal transform needs the observer's latitude
// and longitude, and the sidereal time is the Earth's rotation angle. The
// altitude is geometric and does not include atmospheric refraction. Use the
// returned Horizontal's AboveHorizon method to test visibility.
//
// Accuracy is amateur, arcminute-class: precession is applied, while nutation,
// aberration, and the star's proper motion are below that floor and are not
// modeled. It returns a go-apperr coded error when the observer's latitude or
// longitude is out of range.
func StarPosition(s star.Star, obs astronomy.Observer, t time.Time) (astronomy.Horizontal, error) {
	if err := validateObserver(obs); err != nil {
		return astronomy.Horizontal{}, err
	}

	epochOfDate := julian.JulianYear(t)
	eq := coordinates.PrecessEquatorial(
		coordinates.Equatorial{RA: s.RA, Dec: s.Dec},
		starEpoch, epochOfDate,
	)

	gst := julian.GreenwichSiderealTime(t)
	hz := coordinates.EquatorialToHorizontal(eq, obs.Lat, obs.Lng, gst)

	return astronomy.Horizontal{
		Altitude: hz.Altitude,
		Azimuth:  hz.Azimuth,
	}, nil
}
