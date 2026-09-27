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

import "errors"

// Sentinel causes, each returned by the planet packages inside a go-apperr
// coded error.
var (
	// ErrInvalidLatitude is the cause when the observer's latitude is outside
	// [-90, 90]; the coded error carries astronomy.CodeInvalidLatitude.
	ErrInvalidLatitude = errors.New("planet: observer latitude out of range")
	// ErrInvalidLongitude is the cause when the observer's longitude is
	// outside [-180, 180]; the coded error carries astronomy.CodeInvalidLongitude.
	ErrInvalidLongitude = errors.New("planet: observer longitude out of range")
)
