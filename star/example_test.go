package star_test

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
	"math"
	"time"

	"github.com/Bugs5382/go-astronomy/star"
)

// Barnard's Star crosses the sky faster than any other: its place on the
// J2000 equator in 2000 and a century later, and how far it moved.
func ExampleStar_PositionAt() {
	s, _ := star.GetNamedStar("Barnard's Star")
	fmt.Printf("motion %.2f and %.2f mas/yr, %.0f km/s\n", s.PMRA, s.PMDec, s.RadialVelocity)
	for _, year := range []int{2000, 2100} {
		ra, dec := s.PositionAt(time.Date(year, 1, 1, 12, 0, 0, 0, time.UTC))
		fmt.Printf("%d: RA %.5f Dec %.5f\n", year, ra, dec)
	}
	ra0, dec0 := s.PositionAt(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC))
	ra1, dec1 := s.PositionAt(time.Date(2100, 1, 1, 12, 0, 0, 0, time.UTC))
	moved := math.Hypot((ra1-ra0)*math.Cos(dec0*math.Pi/180), dec1-dec0) * 60
	fmt.Printf("a century moves it %.1f arc minutes\n", moved)
	// Output:
	// motion -797.84 and 10326.93 mas/yr, -111 km/s
	// 2000: RA 269.45208 Dec 4.69339
	// 2100: RA 269.42969 Dec 4.98204
	// a century moves it 17.4 arc minutes
}
