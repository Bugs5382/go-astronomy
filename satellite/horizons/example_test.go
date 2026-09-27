package horizons_test

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
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite/horizons"
)

// standIn serves the recorded 30-day JWST window for every request.
func standIn() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := os.ReadFile("testdata/jwst-window.txt")
		_, _ = w.Write(b)
	}))
}

// The James Webb Space Telescope from Greenwich: one request for the window,
// then everything is interpolated locally.
func ExampleClient_Position() {
	api := standIn()
	defer api.Close()
	c := horizons.New(horizons.WithBaseURL(api.URL))
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005}
	p, err := c.Position(context.Background(), "-170", greenwich, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.4f, azimuth %.4f\n", p.Altitude, p.Azimuth)
	fmt.Printf("RA %.5f, Dec %.5f, range %.0f km\n", p.RA, p.Dec, p.RangeKm)
	// Output:
	// altitude 56.5685, azimuth 156.4667
	// RA 23.26297, Dec 19.77391, range 1294229 km
}

// The geocentric place, shared by every observer.
func ExampleClient_Geocentric() {
	api := standIn()
	defer api.Close()
	c := horizons.New(horizons.WithBaseURL(api.URL))
	g, err := c.Geocentric(context.Background(), "-170", time.Date(2026, 9, 20, 12, 30, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("RA %.5f, Dec %.5f, %.0f km\n", g.RA, g.Dec, g.DistanceKm)
	// Output:
	// RA 359.88542, Dec 13.78232, 1227623 km
}
