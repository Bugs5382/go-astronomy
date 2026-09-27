// Package elevation defines how an observer's height above sea level is looked
// up, so that go-astronomy's rise and set times can take the horizon dip.
//
// The core packages never reach the network: a caller that already knows its
// height sets astronomy.Observer.Elevation and imports nothing from here. A
// caller that wants the height looked up from a coordinate uses a Lookup, such
// as the Open-Meteo adapter in elevation/openmeteo, and Fill copies the result
// into the observer.
//
// Elevation does not change, so a looked-up value can be kept indefinitely. An
// adapter may cache it in process; a durable cache shared across processes is
// the consumer's to provide, keyed however it likes.
package elevation

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
	"context"
	"errors"
	"fmt"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// Lookup returns the height above sea level, in metres, of the ground at a
// coordinate. Latitude is in degrees positive north and longitude in degrees
// positive east. Implementations must be safe for concurrent use.
type Lookup interface {
	Elevation(ctx context.Context, lat, lng float64) (float64, error)
}

// Sentinel causes. Each is returned inside a go-apperr coded error, so
// errors.Is matches these values and apperr.Code recovers the stable code.
var (
	// ErrInvalidLatitude is the cause when a latitude is outside [-90, 90]; the
	// coded error carries astronomy.CodeInvalidLatitude.
	ErrInvalidLatitude = errors.New("elevation: latitude out of range")
	// ErrInvalidLongitude is the cause when a longitude is outside
	// [-180, 180]; the coded error carries astronomy.CodeInvalidLongitude.
	ErrInvalidLongitude = errors.New("elevation: longitude out of range")
	// ErrLookupFailed is the cause when a lookup could not produce a height:
	// the service was unreachable, answered with an error, or returned a
	// value that is not a usable elevation. The coded error carries
	// astronomy.CodeElevationLookup.
	ErrLookupFailed = errors.New("elevation: lookup failed")
)

// ValidateCoordinate returns the coded error for an out-of-range coordinate,
// or nil. Lookup implementations call it before they query anything.
func ValidateCoordinate(lat, lng float64) error {
	if !(lat >= -90 && lat <= 90) {
		return apperr.Coded(astronomy.CodeInvalidLatitude, ErrInvalidLatitude)
	}
	if !(lng >= -180 && lng <= 180) {
		return apperr.Coded(astronomy.CodeInvalidLongitude, ErrInvalidLongitude)
	}
	return nil
}

// LookupFailed wraps a lookup failure in the coded error for
// astronomy.CodeElevationLookup, keeping both ErrLookupFailed and cause
// matchable with errors.Is.
func LookupFailed(cause error) error {
	return apperr.Coded(astronomy.CodeElevationLookup, fmt.Errorf("%w: %w", ErrLookupFailed, cause))
}

// Fill returns obs with Elevation set from the lookup at the observer's
// coordinate. The observer is otherwise unchanged. A height outside the range
// the observer accepts is reported as ErrLookupFailed rather than passed on.
func Fill(ctx context.Context, l Lookup, obs astronomy.Observer) (astronomy.Observer, error) {
	if err := ValidateCoordinate(obs.Lat, obs.Lng); err != nil {
		return obs, err
	}
	h, err := l.Elevation(ctx, obs.Lat, obs.Lng)
	if err != nil {
		return obs, err
	}
	if !earth.ValidElevation(h) {
		return obs, LookupFailed(fmt.Errorf("height %v m out of range", h))
	}
	obs.Elevation = h
	return obs, nil
}
