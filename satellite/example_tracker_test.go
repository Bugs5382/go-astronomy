package satellite_test

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
)

// A tracker for any satellite, from the caller's own element set.
func ExampleNewTracker() {
	e, err := satellite.ParseTLE(exampleLine1, exampleLine2)
	if err != nil {
		panic(err)
	}
	tr := satellite.NewTracker(25544, "ISS (ZARYA)", satellite.ISSStandardMagnitude, satellite.StaticElements(e))
	l, err := tr.Position(context.Background(), astronomy.Observer{Lat: -33.87, Lng: 151.21}, time.Date(2026, 9, 26, 4, 5, 3, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s altitude %.2f, magnitude %.1f\n", tr.Name(), l.Altitude, tr.Magnitude(l))
	// Output:
	// ISS (ZARYA) altitude 88.51, magnitude -1.5
}

// The default cache, with an expiry.
func ExampleNewMemoryCache() {
	c := satellite.NewMemoryCache()
	_ = c.Set(context.Background(), "k", []byte("v"), time.Hour)
	v, ok, _ := c.Get(context.Background(), "k")
	fmt.Println(string(v), ok)
	// Output:
	// v true
}
