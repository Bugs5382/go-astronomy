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

// The element set, and propagating it to an instant.

import (
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
)

// Elements is a parsed two-line element set together with the initialised
// SGP4 propagator state. Angles are in degrees as printed in the TLE; the
// propagator works from its own radian copies.
type Elements struct {
	CatalogNumber    string  // catalog number as printed, possibly Alpha-5 ("A0001")
	SatNum           int     // numeric catalog number, Alpha-5 decoded (A0001 is 100001)
	Classification   byte    // 'U', 'C' or 'S'
	IntlDesignator   string  // international designator, such as "98067A"
	EpochYear        int     // four-digit epoch year (57-99 is 1900s, 00-56 is 2000s)
	EpochDay         float64 // day of year with fraction, 1.0 is 1 January 0h
	NDot             float64 // first derivative of mean motion / 2, rev/day^2
	NDDot            float64 // second derivative of mean motion / 6, rev/day^3
	BStar            float64 // drag term, 1/earth radii
	EphemerisType    int     // usually 0
	ElementSetNumber int
	Inclination      float64 // degrees
	RAAN             float64 // right ascension of the ascending node, degrees
	Eccentricity     float64
	ArgPerigee       float64 // argument of perigee, degrees
	MeanAnomaly      float64 // degrees
	MeanMotion       float64 // revolutions per day
	RevNumber        int     // revolution number at epoch

	epoch time.Time
	rec   *satrec // read-only after initialisation
}

// Epoch returns the element set epoch in UTC.
func (e Elements) Epoch() time.Time { return e.epoch }

// Age returns how far t is from the element set epoch. It is negative when
// t is before the epoch.
func (e Elements) Age(t time.Time) time.Duration { return t.Sub(e.epoch) }

// Propagate returns the TEME position (km) and velocity (km/s) at t.
func (e Elements) Propagate(t time.Time) (posKm, velKmS [3]float64, err error) {
	if e.rec == nil {
		return posKm, velKmS, ErrNotInitialised
	}
	// Whole seconds and nanoseconds separately, so very long spans do not
	// overflow time.Duration.
	sec := t.Unix() - e.epoch.Unix()
	nsec := int64(t.Nanosecond()) - int64(e.epoch.Nanosecond())
	tsince := float64(sec)/60.0 + float64(nsec)/6e10
	return e.PropagateMinutes(tsince)
}

// PropagateMinutes returns the TEME position (km) and velocity (km/s) at
// tsince minutes from the epoch. This is the form the verification data
// uses.
//
// Errors are go-apperr coded with astronomy.CodeSatellitePropagation and wrap
// a *PropagationError (errors.As) and its sentinel (errors.Is). For a decayed
// satellite the position and velocity are still returned with the error, as
// the reference code does.
func (e Elements) PropagateMinutes(tsince float64) (posKm, velKmS [3]float64, err error) {
	if e.rec == nil {
		return posKm, velKmS, apperr.Coded(astronomy.CodeSatellitePropagation, ErrNotInitialised)
	}
	posKm, velKmS, err = sgp4(e.rec, tsince)
	if err != nil {
		err = apperr.Coded(astronomy.CodeSatellitePropagation, err)
	}
	return posKm, velKmS, err
}
