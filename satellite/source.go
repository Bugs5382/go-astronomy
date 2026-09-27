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

// Where element sets come from. The engine never fetches one: a caller passes
// a source explicitly, either its own element sets (StaticElements) or a
// fetcher such as satellite/celestrak.

import (
	"context"
	"errors"
	"fmt"
)

// ErrNoElements is the cause when a source has no element set for the
// catalogue number asked for.
var ErrNoElements = errors.New("satellite: no element set for the catalogue number")

// ElementSource supplies the current element set for a NORAD catalogue
// number. An error comes with no set. Implementations must be safe for
// concurrent use.
type ElementSource interface {
	Elements(ctx context.Context, catalog int) (Elements, error)
}

type staticSource map[int]Elements

// StaticElements returns a source over element sets the caller already has,
// from ParseTLE or ParseOMM, keyed by their catalogue numbers. A later set
// with the same number replaces an earlier one. It never reaches the network.
func StaticElements(sets ...Elements) ElementSource {
	s := staticSource{}
	for _, e := range sets {
		s[e.SatNum] = e
	}
	return s
}

func (s staticSource) Elements(ctx context.Context, catalog int) (Elements, error) {
	if err := ctx.Err(); err != nil {
		return Elements{}, err
	}
	e, ok := s[catalog]
	if !ok {
		return Elements{}, fmt.Errorf("%w: %d", ErrNoElements, catalog)
	}
	return e, nil
}
