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

// Propagation errors: the sentinels and the PropagationError that carries
// the reference code's details.

import (
	"errors"
	"fmt"
)

// Propagation errors. The reference code reports these as numeric codes on
// the satellite record; here each code has a sentinel error, and Propagate
// returns a *PropagationError that wraps it.
var (
	// ErrEccentricity is code 1: the mean eccentricity left the range
	// -0.001 <= e < 1.
	ErrEccentricity = errors.New("satellite: mean eccentricity out of range")
	// ErrMeanMotion is code 2: the mean motion dropped to zero or below.
	ErrMeanMotion = errors.New("satellite: mean motion is not positive")
	// ErrPerturbedEccentricity is code 3: the eccentricity after the
	// lunar-solar periodics left the range 0 <= e <= 1.
	ErrPerturbedEccentricity = errors.New("satellite: perturbed eccentricity out of range")
	// ErrSemiLatusRectum is code 4: the semi-latus rectum went negative.
	ErrSemiLatusRectum = errors.New("satellite: semi-latus rectum is negative")
	// ErrDecayed is code 6: the computed radius is below the Earth's
	// surface. Position and velocity are still returned with this error.
	ErrDecayed = errors.New("satellite: satellite has decayed")
	// ErrNotInitialised is returned when propagating a zero Elements value
	// that did not come from ParseTLE.
	ErrNotInitialised = errors.New("satellite: elements not initialised")
)

// PropagationError describes a failed propagation. Code is the numeric
// error code of the reference implementation (1, 2, 3, 4 or 6).
type PropagationError struct {
	Code    int     // reference error code
	Minutes float64 // minutes since epoch of the failing request
	Value   float64 // the offending quantity (eccentricity, radius and so on)
	err     error
}

func (e *PropagationError) Error() string {
	var what string
	switch e.Code {
	case 1:
		what = fmt.Sprintf("mean eccentricity %f not within range 0.0 <= e < 1.0", e.Value)
	case 2:
		what = fmt.Sprintf("mean motion %f is less than zero", e.Value)
	case 3:
		what = fmt.Sprintf("perturbed eccentricity %f not within range 0.0 <= e <= 1.0", e.Value)
	case 4:
		what = fmt.Sprintf("semilatus rectum %f is less than zero", e.Value)
	case 6:
		what = fmt.Sprintf("mrt %f is less than 1.0 indicating the satellite has decayed", e.Value)
	default:
		what = fmt.Sprintf("error code %d", e.Code)
	}
	return fmt.Sprintf("satellite: %s (%.6f minutes from epoch)", what, e.Minutes)
}

// Unwrap returns the sentinel error for the code, so errors.Is works.
func (e *PropagationError) Unwrap() error { return e.err }

func propErr(code int, tsince, value float64) error {
	var sentinel error
	switch code {
	case 1:
		sentinel = ErrEccentricity
	case 2:
		sentinel = ErrMeanMotion
	case 3:
		sentinel = ErrPerturbedEccentricity
	case 4:
		sentinel = ErrSemiLatusRectum
	case 6:
		sentinel = ErrDecayed
	}
	return &PropagationError{Code: code, Minutes: tsince, Value: value, err: sentinel}
}
