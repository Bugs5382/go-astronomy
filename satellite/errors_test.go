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

import (
	"errors"
	"testing"
	"time"
)

// The error cases must fail the same way whatever the entry path.
func TestErrorSentinels(t *testing.T) {
	tles := readVerTLE(t)
	cases := []struct {
		idx      int
		sentinel error
	}{
		{11, ErrEccentricity},
		{25, ErrDecayed},
		{29, ErrSemiLatusRectum},
		{30, ErrPerturbedEccentricity},
	}
	for _, c := range cases {
		e, err := parseTLE(tles[c.idx].line1, tles[c.idx].line2, false)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = e.PropagateMinutes(verErrors[c.idx].tsince)
		if !errors.Is(err, c.sentinel) {
			t.Errorf("satellite %s: got %v, want %v", e.CatalogNumber, err, c.sentinel)
		}
	}

	var zero Elements
	if _, _, err := zero.Propagate(time.Now()); !errors.Is(err, ErrNotInitialised) {
		t.Errorf("zero Elements: got %v", err)
	}
}
