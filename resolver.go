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

// Resolving an observer's height. Three ways to give an Observer its height:
//
//  1. Leave Height out: the observer is at sea level, and nothing reaches the
//     network.
//  2. Set it by hand: Observer{Lat: 40.71, Lng: -74.01, Height: Feet(5000)}.
//  3. Look it up with an ElevationResolver: ResolveObserverWith, or
//     openmeteo.ResolveObserver for the Open-Meteo default. Resolvers compose
//     with ChainElevation, so a value the caller already has can be tried
//     before a network lookup.
//
// The calculation functions never look anything up; only a resolver does, and
// the only one that reaches the network lives in the openmeteo package.

import "context"

// Sources an ElevationResolver reports for the height it returns.
const (
	// SourceCaller is a height the caller already had (CallerElevation).
	SourceCaller = "caller"
	// SourceStatic is a fixed, configured height (StaticElevation).
	SourceStatic = "static"
	// SourceOpenMeteo is a height from the Open-Meteo elevation API.
	SourceOpenMeteo = "open-meteo"
	// SourceSeaLevel means no resolver produced a height, so the answer is
	// sea level.
	SourceSeaLevel = "sea-level"
)

// ElevationResolver finds the height of the ground at a coordinate (degrees,
// latitude positive north and longitude positive east).
//
// Elevation returns the height and the source that produced it. A resolver
// that cannot produce a height, for any reason, returns SeaLevel and
// SourceSeaLevel instead of an error: a missing height is never fatal. The
// error is only for a cancelled or expired context, which it returns with
// SeaLevel. Implementations must be safe for concurrent use.
type ElevationResolver interface {
	Elevation(ctx context.Context, lat, lng float64) (Height, string, error)
}

// ctxErr returns the sea-level answer for a context that is already done.
func ctxErr(ctx context.Context) (Height, string, error) {
	return SeaLevel, SourceSeaLevel, ctx.Err()
}

type staticResolver struct{ h Height }

// StaticElevation returns a resolver that always answers h, with
// SourceStatic, whatever the coordinate: for a fixed installation whose
// height is configured. A NaN or infinite h is no answer, so the resolver
// reports sea level.
func StaticElevation(h Height) ElevationResolver { return staticResolver{h} }

func (s staticResolver) Elevation(ctx context.Context, _, _ float64) (Height, string, error) {
	if ctx.Err() != nil {
		return ctxErr(ctx)
	}
	if s.h.Err() != nil {
		return SeaLevel, SourceSeaLevel, nil
	}
	return s.h, SourceStatic, nil
}

type callerResolver struct {
	h  Height
	ok bool
}

// CallerElevation returns a resolver for a height the caller may already
// have, such as the ground elevation in a weather reading: with ok set it
// answers h with SourceCaller, and with ok unset it reports sea level so a
// chain moves on to the next resolver.
func CallerElevation(h Height, ok bool) ElevationResolver { return callerResolver{h, ok} }

func (c callerResolver) Elevation(ctx context.Context, _, _ float64) (Height, string, error) {
	if ctx.Err() != nil {
		return ctxErr(ctx)
	}
	if !c.ok || c.h.Err() != nil {
		return SeaLevel, SourceSeaLevel, nil
	}
	return c.h, SourceCaller, nil
}

type chainResolver []ElevationResolver

// ChainElevation returns a resolver that asks each of resolvers in order and
// returns the first answer that is not SourceSeaLevel, with that resolver's
// source. When every resolver reports sea level, so does the chain; it only
// falls back at the end. A done context stops the chain with its error.
func ChainElevation(resolvers ...ElevationResolver) ElevationResolver {
	return chainResolver(append([]ElevationResolver(nil), resolvers...))
}

func (c chainResolver) Elevation(ctx context.Context, lat, lng float64) (Height, string, error) {
	for _, r := range c {
		if ctx.Err() != nil {
			return ctxErr(ctx)
		}
		h, src, err := r.Elevation(ctx, lat, lng)
		if err != nil {
			return SeaLevel, SourceSeaLevel, err
		}
		if src != SourceSeaLevel {
			return h, src, nil
		}
	}
	if ctx.Err() != nil {
		return ctxErr(ctx)
	}
	return SeaLevel, SourceSeaLevel, nil
}

// ResolveObserverWith returns an observer at the coordinate with its height
// from r, and the source that produced it. The time zone is left nil (UTC);
// set TZ on the result. The error is only for a cancelled or expired context;
// every lookup failure falls back to sea level with SourceSeaLevel.
func ResolveObserverWith(ctx context.Context, r ElevationResolver, lat, lng float64) (Observer, string, error) {
	h, src, err := r.Elevation(ctx, lat, lng)
	return Observer{Lat: lat, Lng: lng, Height: h}, src, err
}
