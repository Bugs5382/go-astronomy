package earth_test

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
	"fmt"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/star"
)

// A whole sky at one instant: one StarField shares the instant's precession,
// nutation, and aberration across every star.
func ExampleNewStarField() {
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944}
	field, err := earth.NewStarField(obs, time.Date(2027, 1, 15, 3, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	up := 0
	for _, s := range star.Brighter(4) {
		if field.Position(s).AboveHorizon() {
			up++
		}
	}
	sirius, _ := star.GetNamedStar("Sirius")
	hz := field.Position(sirius)
	fmt.Printf("%d stars of magnitude 4 or brighter are up\n", up)
	fmt.Printf("Sirius at altitude %.4f, azimuth %.4f\n", hz.Altitude, hz.Azimuth)
	// Output:
	// 241 stars of magnitude 4 or brighter are up
	// Sirius at altitude 30.6299, azimuth 161.9241
}
