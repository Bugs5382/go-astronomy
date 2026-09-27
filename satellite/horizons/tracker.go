package horizons

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// Tracker follows one Horizons target. satellite/jwst and satellite/roman
// build one; NewTracker builds one for any other target.
type Tracker struct {
	target, name string
	c            *Client
}

// NewTracker returns a tracker for the Horizons target (a command such as
// "-170") with the given name, fetching through c.
func NewTracker(target, name string, c *Client) *Tracker {
	return &Tracker{target: target, name: name, c: c}
}

// Target returns the Horizons command for the object.
func (t *Tracker) Target() string { return t.target }

// Name returns the object's name.
func (t *Tracker) Name() string { return t.name }

// Position returns the object as the observer sees it at at.
func (t *Tracker) Position(ctx context.Context, obs astronomy.Observer, at time.Time) (Position, error) {
	return t.c.Position(ctx, t.target, obs, at)
}

// Geocentric returns the object's geocentric apparent place at at.
func (t *Tracker) Geocentric(ctx context.Context, at time.Time) (Geocentric, error) {
	return t.c.Geocentric(ctx, t.target, at)
}
