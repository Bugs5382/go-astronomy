package elevation_test

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

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/elevation"
)

// fixed is a Lookup that returns one height everywhere: a stand-in for a real
// service or your own DEM tiles.
type fixed float64

func (f fixed) Elevation(ctx context.Context, lat, lng float64) (float64, error) {
	if err := elevation.ValidateCoordinate(lat, lng); err != nil {
		return 0, err
	}
	return float64(f), nil
}

// Fill copies a looked-up height into the observer.
func ExampleFill() {
	obs, err := elevation.Fill(context.Background(), fixed(1609), astronomy.Observer{Lat: 39.74, Lng: -104.99})
	if err != nil {
		panic(err)
	}
	fmt.Println(obs.Elevation, "m")

	// A reply outside the range an observer accepts is refused.
	_, err = elevation.Fill(context.Background(), fixed(20000), astronomy.Observer{Lat: 1, Lng: 1})
	fmt.Println(err != nil)
	// Output:
	// 1609 m
	// true
}
