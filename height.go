package astronomy

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
	"strconv"

	apperr "github.com/Bugs5382/go-apperr"
)

// MetersPerFoot is the international foot, exactly.
const MetersPerFoot = 0.3048

// Height is a height above sea level. Build one with Meters or Feet; the zero
// value is SeaLevel. Any real value is valid as given: below sea level (the
// Dead Sea shore is -430 m), on a mountain, or in flight. Only NaN and the
// infinities are invalid (see Err), and the functions that take an observer
// reject them with ErrInvalidHeight.
//
// A Height is stored in metres, the unit every calculation uses.
type Height struct {
	m float64
}

// SeaLevel is the zero Height, and the height of an Observer that gives none.
var SeaLevel = Height{}

// Meters returns the height of m metres above sea level.
func Meters(m float64) Height { return Height{m: m} }

// Feet returns the height of ft feet above sea level (one foot is 0.3048 m).
func Feet(ft float64) Height { return Height{m: ft * MetersPerFoot} }

// Meters returns the height in metres.
func (h Height) Meters() float64 { return h.m }

// Feet returns the height in feet.
func (h Height) Feet() float64 { return h.m / MetersPerFoot }

// String formats the height in metres, for example "1524 m".
func (h Height) String() string {
	return strconv.FormatFloat(h.m, 'f', -1, 64) + " m"
}

// ErrInvalidHeight is the cause when a Height is NaN or infinite; the coded
// error carries CodeInvalidHeight.
var ErrInvalidHeight = errors.New("astronomy: height is not a finite number")

// Err returns the coded ErrInvalidHeight for a NaN or infinite height, and nil
// for every real value.
func (h Height) Err() error {
	if math.IsNaN(h.m) || math.IsInf(h.m, 0) {
		return apperr.Coded(CodeInvalidHeight, ErrInvalidHeight)
	}
	return nil
}
