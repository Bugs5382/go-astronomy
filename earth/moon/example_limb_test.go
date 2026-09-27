package moon_test

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

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// Which way the crescent faces for an observer in Denver on a June evening,
// two hours after sunset, with the Moon low in the west.
func ExampleBrightLimbAt() {
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	when := time.Date(2027, 6, 8, 4, 0, 0, 0, time.UTC) // 22:00 MDT on the 7th

	pos, err := moon.Position(obs, when)
	if err != nil {
		panic(err)
	}
	bl, err := moon.BrightLimbAt(obs, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Moon at altitude %.1f, azimuth %.1f, %.0f%% lit\n", pos.Altitude, pos.Azimuth, 100*moon.Illumination(when))
	fmt.Printf("position angle %.1f from celestial north\n", bl.PositionAngle)
	fmt.Printf("parallactic angle %.1f\n", bl.Parallactic)
	fmt.Printf("zenith angle %.1f from up on the screen\n", bl.ZenithAngle)
	// Output:
	// Moon at altitude 15.3, azimuth 283.0, 16% lit
	// position angle 283.2 from celestial north
	// parallactic angle 52.7
	// zenith angle 230.5 from up on the screen
}

// Turning ZenithAngle into a drawing direction: with the top of the screen as
// the observer's up, the angle turns counter-clockwise, toward the left.
func ExampleBrightLimb_screen() {
	bl, err := moon.BrightLimbAt(astronomy.Observer{Lat: 39.74, Lng: -104.99}, time.Date(2027, 6, 8, 4, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	// Screen coordinates: x to the right, y down. The unit vector toward the
	// bright limb is (-sin a, -cos a) for a counter-clockwise angle a from up.
	x, y := screenDir(bl.ZenithAngle)
	fmt.Printf("lit side toward x=%.2f y=%.2f\n", x, y)
	// Output:
	// lit side toward x=0.77 y=0.64
}

func screenDir(deg float64) (x, y float64) {
	r := deg * math.Pi / 180
	return -math.Sin(r), -math.Cos(r)
}
