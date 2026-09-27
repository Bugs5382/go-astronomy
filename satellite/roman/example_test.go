package roman_test

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
	"github.com/Bugs5382/go-astronomy/satellite/roman"
)

// Where the telescope is in Sydney's sky, from a stand-in serving the
// recorded Horizons window. In production, pass horizons.New().
func ExampleNew() {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := os.ReadFile("testdata/roman-window-full.txt")
		_, _ = w.Write(b)
	}))
	defer api.Close()

	tracker := roman.New(horizons.New(horizons.WithBaseURL(api.URL)))
	sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	p, err := tracker.Position(context.Background(), sydney, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: altitude %.2f, azimuth %.2f, %.0f km away\n", tracker.Name(), p.Altitude, p.Azimuth, p.RangeKm)
	// Output:
	// Nancy Grace Roman Space Telescope: altitude 69.12, azimuth 38.77, 1232376 km away
}
