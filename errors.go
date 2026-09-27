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
	"context"
	"fmt"
	"sync"

	apperr "github.com/Bugs5382/go-apperr"
	log "github.com/Bugs5382/go-log"
)

// ErrorServiceDigit is the leading digit that every go-astronomy error code
// shares, so a code is attributable to this module at a glance. It is the
// go-apperr service prefix for the module's code namespace.
const ErrorServiceDigit = 7

// Stable numeric error codes for the module. Each error-returning function wraps
// its cause with one of these via go-apperr, so a caller can recover the code
// with apperr.Code and branch on it without matching error strings. The values
// are part of the public contract: never renumber a code, only add new ones.
const (
	// CodeInvalidLatitude marks an observer latitude outside [-90, 90].
	CodeInvalidLatitude = 7001
	// CodeInvalidLongitude marks an observer longitude outside [-180, 180].
	CodeInvalidLongitude = 7002
	// CodeInvalidSegmentation marks a Segmentation whose levels are empty or not
	// in strictly ascending altitude order.
	CodeInvalidSegmentation = 7003
	// CodeInvalidMagnitudeRange marks a stellar magnitude query whose bright
	// bound is fainter than its faint bound, which can select no star.
	CodeInvalidMagnitudeRange = 7004
	// CodeInvalidCanvas marks a screen projection whose canvas width or height is
	// not positive, so it spans no pixels.
	CodeInvalidCanvas = 7005
	// CodeInvalidHorizonFraction marks a viewport whose horizon fraction is
	// outside [0, 1], so the horizon line would fall off the canvas.
	CodeInvalidHorizonFraction = 7006
	// CodeInvalidPeakAltitude marks a NormalizedByPeak projection whose peak
	// altitude is not positive, which cannot normalize an arch.
	CodeInvalidPeakAltitude = 7007
	// CodeInvalidFieldOfView marks an azimuth-based projection whose field of view
	// (Directional) or sunrise-to-sunset azimuth span (FullArc) is not positive.
	CodeInvalidFieldOfView = 7008
	// CodeInvalidElevationScale marks a Geometric projection whose elevation
	// degrees-per-pixel is not positive.
	CodeInvalidElevationScale = 7009
	// CodeXModeNotAzimuth marks a column-to-azimuth inverse query on a projection
	// whose horizontal mode is not azimuth-based, so no azimuth axis exists.
	CodeXModeNotAzimuth = 7010
	// CodeInvalidHeight marks an observer Height that is NaN or infinite.
	CodeInvalidHeight = 7011
	// CodeInvalidElements marks a satellite element set (TLE or OMM) that
	// could not be parsed.
	CodeInvalidElements = 7014
	// CodeSatellitePropagation marks an element set SGP4 could not propagate
	// to the requested instant, such as a satellite that has decayed.
	CodeSatellitePropagation = 7015
	// CodeInvalidPassWindow marks a pass search whose end is not after its
	// start, or which spans more than 31 days.
	CodeInvalidPassWindow = 7016
)

// errorEntries is the module's code table. It feeds the go-apperr registry that
// Errors returns and, through Registry.Markdown, an error-codes reference.
var errorEntries = []apperr.Entry{
	{Code: CodeInvalidLatitude, Title: "observer", Cause: "observer latitude outside [-90, 90]"},
	{Code: CodeInvalidLongitude, Title: "observer", Cause: "observer longitude outside [-180, 180]"},
	{Code: CodeInvalidSegmentation, Title: "segmentation", Cause: "segmentation levels empty or not strictly ascending"},
	{Code: CodeInvalidMagnitudeRange, Title: "magnitude", Cause: "magnitude range bright bound fainter than faint bound"},
	{Code: CodeInvalidCanvas, Title: "projection", Cause: "canvas width or height not positive"},
	{Code: CodeInvalidHorizonFraction, Title: "projection", Cause: "horizon fraction outside [0, 1]"},
	{Code: CodeInvalidPeakAltitude, Title: "projection", Cause: "normalized peak altitude not positive"},
	{Code: CodeInvalidFieldOfView, Title: "projection", Cause: "azimuth field of view or arc span not positive"},
	{Code: CodeInvalidElevationScale, Title: "projection", Cause: "geometric elevation degrees-per-pixel not positive"},
	{Code: CodeXModeNotAzimuth, Title: "projection", Cause: "column-to-azimuth query on a non-azimuth horizontal mode"},
	{Code: CodeInvalidHeight, Title: "observer", Cause: "observer height is NaN or infinite"},
	{Code: CodeInvalidElements, Title: "satellite", Cause: "satellite element set could not be parsed"},
	{Code: CodeSatellitePropagation, Title: "satellite", Cause: "SGP4 could not propagate the element set to the instant"},
	{Code: CodeInvalidPassWindow, Title: "satellite", Cause: "pass window end not after start, or longer than 31 days"},
}

// logSink adapts a go-log Logger to the go-apperr Logger interface. It is the
// pluggable logging sink go-apperr calls from Registry.PresentContext; the
// library never invokes that path itself, so it stays silent by default. When a
// consumer does present a coded error with a context, the sink derives a logger
// correlated with any OpenTelemetry span already on the context. It never starts
// a tracer or exporter: OpenTelemetry is a transitive dependency of go-log that
// remains dormant unless the surrounding service turns it on.
type logSink struct{ l log.Logger }

// LogCoded emits one structured error line for the code, using go-log's
// context-aware logger so an active trace is correlated when the service has set
// one up.
func (s logSink) LogCoded(ctx context.Context, code int, err error) {
	s.l.Ctx(ctx).Error(err, "coded error", log.F("code", code))
}

// errorRegistry builds the module's go-apperr registry exactly once, on first
// use. Constructing the go-log logger lazily keeps a plain import of this
// package free of any logging setup, and building it here rather than at init
// time keeps that cost off programs that never present a coded error.
var errorRegistry = sync.OnceValue(func() *apperr.Registry {
	reg, err := apperr.NewRegistry(
		errorEntries,
		apperr.WithService(ErrorServiceDigit),
		apperr.WithLogger(logSink{l: log.NewLogger("go-astronomy")}),
	)
	if err != nil {
		// The entries and service prefix are compile-time constants, so a
		// failure here is a programming error in this file, not a runtime input.
		panic(fmt.Sprintf("astronomy: building error registry: %v", err))
	}
	return reg
})

// Errors returns the module's go-apperr registry: the stable code table plus the
// presentation and logging policy around it. A consumer renders a coded error at
// its edge with Registry.Present (or PresentContext, which also drives the wired
// go-log logger) to obtain a sanitized, quotable message, looks a code up with
// Registry.Describe, or emits the whole table as Markdown with Registry.Markdown.
// The registry is built once and is safe for concurrent use.
func Errors() *apperr.Registry { return errorRegistry() }
