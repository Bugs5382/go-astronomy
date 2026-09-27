package astronomy_test

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
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// sunrise returns the start of the sunrise band on the civil day of date.
func sunrise(obs astronomy.Observer, date time.Time) time.Time {
	day, err := earth.NewSunTimes(obs, date)
	if err != nil {
		panic(err)
	}
	for _, s := range day.Segments() {
		if s.Label == earth.LabelSunrise {
			return s.From
		}
	}
	return time.Time{}
}

// Always sea level: an observer with no height. Nothing is looked up.
func Example_seaLevel() {
	nyc := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC}
	fmt.Println("height:", nyc.Height)
	fmt.Println("sunrise:", sunrise(nyc, time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)).Format("15:04:05 MST"))
	// Output:
	// height: 0 m
	// sunrise: 09:25:00 UTC
}

// By hand: a height in feet or metres, used as given. New York at 5000 ft is
// not the real ground, and that is allowed.
func Example_heightByHand() {
	date := time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)
	nyc := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC}
	high := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC, Height: astronomy.Feet(5000)}
	same := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC, Height: astronomy.Meters(1524)}

	fmt.Println("5000 ft is", high.Height, "and equal to Meters(1524):", high == same)
	fmt.Printf("dip %.3f degrees\n", earth.HorizonDip(high.Height))
	fmt.Printf("sunrise %.1f minutes earlier\n", sunrise(nyc, date).Sub(sunrise(high, date)).Minutes())
	// Output:
	// 5000 ft is 1524 m and equal to Meters(1524): true
	// dip 1.145 degrees
	// sunrise 7.2 minutes earlier
}

// Lookup: a chain tries a height the caller already has, then a fixed value,
// and falls back to sea level only when nothing answers. The Open-Meteo
// lookup (package openmeteo) slots into the same chain.
func Example_lookup() {
	ctx := context.Background()
	weatherHeight, haveIt := astronomy.Meters(0), false // no reading this time

	resolver := astronomy.ChainElevation(
		astronomy.CallerElevation(weatherHeight, haveIt),
		astronomy.StaticElevation(astronomy.Meters(21)),
	)
	obs, source, err := astronomy.ResolveObserverWith(ctx, resolver, 40.71, -74.01)
	if err != nil {
		panic(err) // only for a cancelled or expired ctx
	}
	fmt.Println(obs.Height, "from", source)

	obs, source, _ = astronomy.ResolveObserverWith(ctx, astronomy.ChainElevation(), 40.71, -74.01)
	fmt.Println(obs.Height, "from", source)
	// Output:
	// 21 m from static
	// 0 m from sea-level
}

// A moving observer passes its position and height for each instant. Here a
// flight from New York (JFK) to London (LHR) on the June solstice, leaving at
// 00:00 UTC and cruising at 36000 ft: from altitude the Sun clears the dipped
// horizon before it rises for anyone on the ground below.
func Example_flightNYCToLondon() {
	jfk := [2]float64{40.64, -73.78}
	lhr := [2]float64{51.47, -0.45}
	depart := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)
	const flight = 7 * time.Hour

	for _, f := range []float64{0, 0.25, 0.5, 0.7, 0.75, 1} {
		lat, lng := greatCircle(jfk, lhr, f)
		height := astronomy.Feet(36000)
		if f == 0 || f == 1 {
			height = astronomy.SeaLevel // on the runway
		}
		obs := astronomy.Observer{Lat: lat, Lng: lng, Height: height}
		when := depart.Add(time.Duration(f * float64(flight)))

		sun := earth.SunPosition(obs, when)
		// The Sun is up when its centre clears the refracted horizon, which
		// the dip lowers for an observer at height.
		horizon := earth.HorizonAltitude - earth.HorizonDip(obs.Height)
		ground := sun.Altitude > earth.HorizonAltitude
		fmt.Printf("%s %6.2f %7.2f %6.0f ft sun %6.2f up in the air %-5v on the ground %v\n",
			when.Format("15:04"), lat, lng, obs.Height.Feet(), sun.Altitude, sun.Altitude > horizon, ground)
	}
	// Output:
	// 00:00  40.64  -73.78      0 ft sun   3.98 up in the air true  on the ground true
	// 01:45  47.58  -59.32  36000 ft sun -12.86 up in the air false on the ground false
	// 03:30  52.22  -41.30  36000 ft sun -13.76 up in the air false on the ground false
	// 04:54  53.64  -24.90  36000 ft sun  -2.43 up in the air true  on the ground false
	// 05:15  53.64  -20.70  36000 ft sun   1.66 up in the air true  on the ground true
	// 07:00  51.47   -0.45      0 ft sun  26.78 up in the air true  on the ground true
}

// greatCircle returns the point a fraction f of the way from a to b along the
// great circle, as latitude and longitude in degrees.
func greatCircle(a, b [2]float64, f float64) (float64, float64) {
	const r = math.Pi / 180
	la1, lo1, la2, lo2 := a[0]*r, a[1]*r, b[0]*r, b[1]*r
	d := math.Acos(math.Sin(la1)*math.Sin(la2) + math.Cos(la1)*math.Cos(la2)*math.Cos(lo2-lo1))
	s1, s2 := math.Sin((1-f)*d)/math.Sin(d), math.Sin(f*d)/math.Sin(d)
	x := s1*math.Cos(la1)*math.Cos(lo1) + s2*math.Cos(la2)*math.Cos(lo2)
	y := s1*math.Cos(la1)*math.Sin(lo1) + s2*math.Cos(la2)*math.Sin(lo2)
	z := s1*math.Sin(la1) + s2*math.Sin(la2)
	return math.Atan2(z, math.Hypot(x, y)) / r, math.Atan2(y, x) / r
}
