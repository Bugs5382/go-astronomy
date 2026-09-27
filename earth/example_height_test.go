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
)

// The dip of the sea horizon for an observer at Denver's height.
func ExampleHorizonDip() {
	fmt.Printf("%.3f degrees\n", earth.HorizonDip(astronomy.Meters(1609)))
	fmt.Printf("%.3f degrees below sea level\n", earth.HorizonDip(astronomy.Meters(-430)))
	fmt.Printf("%.3f degrees at 36000 ft\n", earth.HorizonDip(astronomy.Feet(36000)))
	// Output:
	// 1.177 degrees
	// 0.000 degrees below sea level
	// 3.073 degrees at 36000 ft
}

// Height moves sunrise: the same day at Denver, at sea level and at the
// city's 1609 m. The dip moves it about 7.3 minutes earlier; the thinner air
// refracts the Sun less and takes back about half a minute.
func Example_observerHeight() {
	sea := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC}
	high := sea
	high.Height = astronomy.Meters(1609)
	date := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)

	sunrise := func(obs astronomy.Observer) time.Time {
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
	a, b := sunrise(sea), sunrise(high)
	fmt.Println("sea level:", a.Format("15:04 MST"))
	fmt.Println("1609 m:   ", b.Format("15:04 MST"))
	fmt.Printf("earlier by %.1f minutes\n", a.Sub(b).Minutes())
	// Output:
	// sea level: 11:32 UTC
	// 1609 m:    11:25 UTC
	// earlier by 6.8 minutes
}

// Twilight stays on the astronomical horizon by default; WithTwilightDip
// moves it with the horizon the observer sees.
func ExampleSegmentation_WithTwilightDip() {
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)}
	date := time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)
	civilDawn := func(seg earth.Segmentation) time.Time {
		day, err := earth.NewSunTimesWith(obs, date, seg)
		if err != nil {
			panic(err)
		}
		for _, s := range day.Segments() {
			if s.Label == earth.LabelCivilDawn {
				return s.From
			}
		}
		return time.Time{}
	}
	usno := civilDawn(earth.DefaultSegmentation)
	dipped := civilDawn(earth.DefaultSegmentation.WithTwilightDip())
	fmt.Printf("civil dawn moves %.1f minutes earlier\n", usno.Sub(dipped).Minutes())
	// Output:
	// civil dawn moves 6.1 minutes earlier
}

// The refraction at the horizon thins with the air: at Denver, at 5000 ft,
// at a cruising 35000 ft, and at 15 km, against sea level.
func ExampleAtmosphere_Factor() {
	for _, h := range []astronomy.Height{astronomy.SeaLevel, astronomy.Meters(1609), astronomy.Feet(5000), astronomy.Feet(35000), astronomy.Meters(15000)} {
		f := earth.StandardAtmosphere.Factor(h)
		fmt.Printf("%-7s factor %.3f, horizon refraction %4.1f arcmin\n", h, f, earth.StandardAtmosphere.Refraction(0, h)*60)
	}
	// Output:
	// 0 m     factor 1.000, horizon refraction 34.5 arcmin
	// 1609 m  factor 0.854, horizon refraction 29.5 arcmin
	// 1524 m  factor 0.862, horizon refraction 29.7 arcmin
	// 10668 m factor 0.311, horizon refraction 10.7 arcmin
	// 15000 m factor 0.159, horizon refraction  5.5 arcmin
}

// From a plane at 35000 ft the dip lowers the horizon by about 3 degrees, and
// the thin air gives less than a third of the sea-level refraction. The Sun is
// up for the plane while its centre is above HorizonAltitudeAt.
func ExampleHorizonAltitudeAt() {
	for _, h := range []astronomy.Height{astronomy.SeaLevel, astronomy.Meters(1609), astronomy.Feet(35000)} {
		f := earth.StandardAtmosphere.Factor(h)
		fmt.Printf("%.0f ft: dip %.3f, refraction lost %.3f, sunrise at %.3f degrees\n",
			h.Feet(), earth.HorizonDip(h), (1-f)*earth.HorizonRefraction, earth.HorizonAltitudeAt(h))
	}
	// Output:
	// 0 ft: dip 0.000, refraction lost 0.000, sunrise at -0.833 degrees
	// 5279 ft: dip 1.177, refraction lost 0.082, sunrise at -1.927 degrees
	// 35000 ft: dip 3.030, refraction lost 0.391, sunrise at -3.472 degrees
}

// With a local reading of station pressure and temperature, the refraction
// follows the measured air instead of the standard atmosphere: a cold winter
// morning in Denver.
func ExampleSegmentation_WithAtmosphere() {
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)}
	date := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	sunrise := func(seg earth.Segmentation) time.Time {
		day, err := earth.NewSunTimesWith(obs, date, seg)
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
	cold := earth.MeasuredAtmosphere(845, -15)
	fmt.Printf("standard air factor %.3f, measured air factor %.3f\n",
		earth.StandardAtmosphere.Factor(obs.Height), cold.Factor(obs.Height))
	fmt.Println("standard air:", sunrise(earth.DefaultSegmentation).Format("15:04:05 MST"))
	fmt.Println("measured air:", sunrise(earth.DefaultSegmentation.WithAtmosphere(cold)).Format("15:04:05 MST"))
	// Output:
	// standard air factor 0.854, measured air factor 0.918
	// standard air: 14:12:54 UTC
	// measured air: 14:12:41 UTC
}
